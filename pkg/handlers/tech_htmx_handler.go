package handlers

import (
	"fmt"
	"time"

	"hvac-system/pkg/broker"

	"github.com/pocketbase/pocketbase/core"
)

// HTMX Handlers - Partial updates without full page reload

// POST /api/tech/job/{id}/status - Update job status with HTMX
// Returns partial HTML to replace the action button section
func (h *TechHandler) UpdateJobStatusHTMX(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	newStatus := e.Request.FormValue("status")

	job, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job not found")
	}

	// Validate status transition
	currentStatus := job.GetString("job_status")
	validTransition := map[string][]string{
		"assigned": {"moving"},
		"moving":   {"working"},
		"working":  {"completed"},
	}

	allowed := false
	if next, ok := validTransition[currentStatus]; ok {
		for _, s := range next {
			if s == newStatus {
				allowed = true
				break
			}
		}
	}

	if !allowed {
		return e.String(400, fmt.Sprintf("Cannot transition from %s to %s", currentStatus, newStatus))
	}

	// Update job status
	job.Set("job_status", newStatus)
	job.Set(fmt.Sprintf("%s_start_at", newStatus), time.Now())

	if err := h.App.Save(job); err != nil {
		return e.String(500, "Failed to update status: "+err.Error())
	}

	// Publish events
	h.Broker.Publish(broker.ChannelCustomer, jobID, broker.Event{
		Type:      "job.status_changed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"status":  newStatus,
			"tech_id": e.Auth.Id,
		},
	})

	h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
		Type:      "job.status_changed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"booking_id": jobID,
			"status":     newStatus,
			"tech_id":    e.Auth.Id,
		},
	})

	// Return updated action button HTML
	e.Response.Header().Set("HX-Trigger", `{"statusUpdated": true}`)
	return h.renderJobActionButtons(e, job)
}

// GET /api/tech/job/{id}/invoice - Get invoice preview
func (h *TechHandler) GetJobInvoice(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")

	job, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job not found")
	}

	// Get invoice
	invoices, err := h.App.FindRecordsByFilter(
		"invoices",
		fmt.Sprintf("booking_id='%s'", jobID),
		"",
		1,
		0,
		nil,
	)

	if len(invoices) == 0 {
		// Generate invoice if not exists
		inv, err := h.InvoiceService.GenerateInvoice(jobID)
		if err != nil {
			return e.String(500, "Failed to generate invoice")
		}
		invoices = []*core.Record{inv}
	}

	data := map[string]interface{}{
		"Job":     job,
		"Invoice": invoices[0],
	}

	// Return invoice HTML snippet
	return h.renderPartial(e, "tech/partials/invoice", data)
}

// POST /api/tech/job/{id}/payment - Process payment
func (h *TechHandler) ProcessPayment(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	paymentMethod := e.Request.FormValue("payment_method")

	invoices, _ := h.App.FindRecordsByFilter(
		"invoices",
		fmt.Sprintf("booking_id='%s'", jobID),
		"",
		1,
		0,
		nil,
	)

	if len(invoices) == 0 {
		return e.String(404, "Invoice not found")
	}

	invoice := invoices[0]
	invoice.Set("payment_method", paymentMethod)
	invoice.Set("status", "paid")

	if err := h.App.Save(invoice); err != nil {
		return e.String(500, "Failed to update payment")
	}

	// Update job status if needed
	job, _ := h.App.FindRecordById("bookings", jobID)
	if job != nil {
		job.Set("payment_status", "paid")
		h.App.Save(job)
	}

	// Publish payment event
	h.Broker.Publish(broker.ChannelCustomer, jobID, broker.Event{
		Type:      "payment.processed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"amount": invoice.GetFloat("total_amount"),
			"method": paymentMethod,
			"status": "paid",
		},
	})

	e.Response.Header().Set("HX-Trigger", `{"paymentComplete": true}`)
	return e.JSON(200, map[string]interface{}{
		"status":  "success",
		"message": "Payment processed successfully",
	})
}

// GET /api/tech/jobs/list - Get refreshed job list (for HTMX pull)
func (h *TechHandler) GetJobsListHTMX(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.String(401, "Unauthorized")
	}

	statusFilter := e.Request.URL.Query().Get("status")
	var filterQuery string

	switch statusFilter {
	case "new":
		// TAB 1: MỚI GIAO (Chỉ việc Admin vừa giao, chưa bấm nhận/di chuyển)
		filterQuery = fmt.Sprintf("technician_id='%s' && job_status='assigned'", authRecord.Id)

	case "active":
		// TAB 2: ĐANG LÀM (Đang di chuyển hoặc Đang sửa tại nhà khách)
		filterQuery = fmt.Sprintf("technician_id='%s' && (job_status='moving' || job_status='working')", authRecord.Id)

	case "completed":
		// TAB 3: HOÀN THÀNH (Lịch sử đã xong)
		filterQuery = fmt.Sprintf("technician_id='%s' && job_status='completed'", authRecord.Id)

	case "all":
		// Tất cả (nếu cần)
		filterQuery = fmt.Sprintf("technician_id='%s'", authRecord.Id)

	default:
		// Mặc định: Hiện các việc chưa xong (Mới + Đang làm) để thợ không bị sót việc
		filterQuery = fmt.Sprintf("technician_id='%s' && (job_status='assigned' || job_status='moving' || job_status='working')", authRecord.Id)
	}

	jobs, err := h.App.FindRecordsByFilter(
		"bookings",
		filterQuery,
		"-booking_time",
		100,
		0,
		nil,
	)

	if err != nil || jobs == nil {
		jobs = []*core.Record{}
	}

	data := map[string]interface{}{
		"Jobs": jobs,
	}

	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	return h.renderPartial(e, "tech/partials/jobs_list", data)
}

// renderJobActionButtons returns the action button HTML based on job status
func (h *TechHandler) renderJobActionButtons(e *core.RequestEvent, job *core.Record) error {
	// Chuẩn bị dữ liệu cho template
	data := map[string]interface{}{
		"Job": job,
	}

	// Gọi file view: views/pages/tech/partials/_job_actions.html
	// Lưu ý: Tên template phải khớp với {{define "tech/partials/job_actions"}}
	// Chúng ta KHÔNG trả về wrapper div nữa, vì frontend dùng hx-swap="innerHTML"
	return h.renderPartial(e, "tech/partials/job_actions", data)
}

// renderPartial renders a partial template and returns HTML content
func (h *TechHandler) renderPartial(e *core.RequestEvent, templateName string, data interface{}) error {
	// [QUAN TRỌNG] Phải Clone trước khi Execute để không làm "bẩn" template gốc
	// Nếu execute trực tiếp h.Templates, các trang khác sẽ không thể Clone được nữa -> Crash app
	tmpl, err := h.Templates.Clone()
	if err != nil {
		fmt.Printf("TEMPLATE CLONE ERROR in Partial: %v\n", err)
		return e.String(500, "Template error")
	}

	// Execute trên bản sao (tmpl) thay vì bản gốc (h.Templates)
	if err := tmpl.ExecuteTemplate(e.Response, templateName, data); err != nil {
		fmt.Printf("TEMPLATE ERROR: %v (Name: %s)\n", err, templateName)
		return e.String(500, "Template error: "+err.Error())
	}
	return nil
}
