package handlers

import (
	"fmt"
	"net/http"
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

	// Validate status transition (Updated to match UI flow)
	currentStatus := job.GetString("job_status")
	validTransition := map[string][]string{
		"pending":  {"moving", "cancelled"},
		"assigned": {"moving", "cancelled"},
		// "assigned": {"moving", "cancelled"},
		"moving":  {"arrived", "working", "cancelled"},
		"arrived": {"working", "cancelled"},
		"working": {"completed", "cancelled"},
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
		if currentStatus == "cancelled" {
			return e.String(409, "Đơn hàng này đã bị hủy hoặc thay đổi trạng thái bởi Admin.")
		}
		return e.String(409, fmt.Sprintf("Trạng thái không hợp lệ (Hiện tại: %s)", currentStatus))
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

// POST /api/tech/job/{id}/payment - Process payment and signature
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

	// Handle Signature Upload
	// Expecting "signature_file" from FormData (converted from canvas blob)
	file, _, err := e.Request.FormFile("signature_file")
	if err == nil {
		defer file.Close()
		// Create a filesystem file from multipart/form-data
		// PocketBase core.FileField expects *multipart.FileHeader or similar.
		// Actually e.FindUploadedFiles is easier.
	}

	files, _ := e.FindUploadedFiles("signature_file")
	if len(files) > 0 {
		invoice.Set("customer_signature", files[0])
	}

	if err := h.App.Save(invoice); err != nil {
		fmt.Printf("Payment Save Error: %v\n", err)
		return e.String(500, "Failed to update payment: "+err.Error())
	}

	// Update job status if needed
	job, _ := h.App.FindRecordById("bookings", jobID)
	if job != nil {
		job.Set("payment_status", "paid")
		// Maybe set job_status to completed if not already (it is)
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

	// HTMX redirect or Client redirect
	// Since this is a full page form usually (or heavy partial), let's redirect to dashboard
	// Or return success JSON
	if e.Request.Header.Get("HX-Request") != "" {
		e.Response.Header().Set("HX-Redirect", "/tech/jobs")
		return e.NoContent(200)
	}

	return e.Redirect(http.StatusSeeOther, "/tech/jobs")
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

	// Prepare View Models using shared logic
	var viewModels []JobViewModel
	for _, job := range jobs {
		vm := JobViewModel{Record: job}

		// Format Time (Reusing logic from TechHandler.JobsList)
		rawTime := job.GetString("booking_time")
		parsedTime, err := time.Parse("2006-01-02 15:04", rawTime)
		if err != nil {
			parsedTime, _ = time.Parse("2006-01-02 15:04:05.000Z", rawTime)
		}

		if !parsedTime.IsZero() {
			endTime := parsedTime.Add(2 * time.Hour)
			vm.DisplayTime = fmt.Sprintf("%02d:%02d - %02d:%02d %02d/%02d",
				parsedTime.Hour(), parsedTime.Minute(),
				endTime.Hour(), endTime.Minute(),
				parsedTime.Day(), parsedTime.Month(),
			)
		} else {
			vm.DisplayTime = "Chưa hẹn giờ"
		}

		viewModels = append(viewModels, vm)
	}

	data := map[string]interface{}{
		"Jobs": viewModels,
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

// POST /api/tech/status/toggle
func (h *TechHandler) ToggleOnlineStatus(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.String(401, "Unauthorized")
	}

	isActive := authRecord.GetBool("active")
	newStatus := !isActive
	authRecord.Set("active", newStatus)

	if err := h.App.Save(authRecord); err != nil {
		return e.String(500, "Failed to update status")
	}

	// CRITICAL FIX: Refresh auth token to update cached "active" value
	// Without this, e.Auth on next request will still have old value from JWT
	newToken, err := authRecord.NewAuthToken()
	if err != nil {
		fmt.Printf("Warning: Failed to refresh auth token: %v\n", err)
	} else {
		http.SetCookie(e.Response, &http.Cookie{
			Name:     "pb_auth",
			Value:    newToken,
			Path:     "/",
			Secure:   false,
			HttpOnly: true,
		})
	}

	return e.JSON(200, map[string]bool{"active": newStatus})
}
