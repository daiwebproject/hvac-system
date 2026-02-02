package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"hvac-system/internal/adapter/repository" // [NEW]
	domain "hvac-system/internal/core"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type TechHandler struct {
	App            *pocketbase.PocketBase
	Templates      *template.Template
	Broker         *broker.SegmentedBroker
	Inventory      *services.InventoryService
	InvoiceService *services.InvoiceService
	BookingService domain.BookingService
	SettingsRepo   *repository.SettingsRepo // [NEW]
}

// --- Auth ---

func (h *TechHandler) ShowLogin(e *core.RequestEvent) error {
	return RenderPage(h.Templates, e, "layouts/auth.html", "tech/login.html", nil)
}

func (h *TechHandler) ProcessLogin(e *core.RequestEvent) error {
	email := e.Request.FormValue("email")
	password := e.Request.FormValue("password")

	record, err := h.App.FindAuthRecordByEmail("technicians", email)
	if err != nil || !record.ValidatePassword(password) {
		return e.JSON(400, map[string]string{"error": "Email hoặc mật khẩu không đúng"})
	}

	token, err := record.NewAuthToken()
	if err != nil {
		return e.String(500, "Lỗi tạo token")
	}

	http.SetCookie(e.Response, &http.Cookie{
		Name:     "pb_auth",
		Value:    token,
		Path:     "/",
		Secure:   false,
		HttpOnly: true,
	})

	return e.Redirect(http.StatusSeeOther, "/tech/jobs")
}

func (h *TechHandler) Logout(e *core.RequestEvent) error {
	http.SetCookie(e.Response, &http.Cookie{
		Name:   "pb_auth",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	return e.Redirect(http.StatusSeeOther, "/tech/login")
}

// --- Mobile Job Views ---

// 1. Cập nhật hàm lấy dữ liệu chung cho Dashboard
func (h *TechHandler) getTechCommonData(techID string) map[string]interface{} {
	// A. Đếm việc đang làm (Active)
	activeRecords, _ := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && (job_status='moving' || job_status='working')", techID),
		"", 0, 0, nil,
	)
	activeCount := len(activeRecords)

	// B. Đếm việc hoàn thành hôm nay
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).Format("2006-01-02 15:04:05")

	completedTodayRecords, _ := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && job_status='completed' && updated >= '%s'", techID, startOfDay),
		"", 0, 0, nil,
	)
	completedTodayCount := len(completedTodayRecords)

	// C. Tính doanh thu (Hôm nay & Tổng cộng)
	// Lưu ý: Cần join với bảng invoices để lấy số tiền thực tế (total_amount)
	// Ở đây dùng cách đơn giản là loop qua danh sách invoice của thợ
	invoices, _ := h.App.FindRecordsByFilter(
		"invoices",
		fmt.Sprintf("booking_id.technician_id='%s' && status='paid'", techID),
		"", 0, 0, nil,
	)

	todayEarnings := 0.0
	totalEarnings := 0.0

	for _, inv := range invoices {
		amount := inv.GetFloat("total_amount")
		totalEarnings += amount

		// Check nếu invoice được tạo hôm nay
		created := inv.GetDateTime("created").Time()
		if created.After(time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())) {
			todayEarnings += amount
		}
	}

	return map[string]interface{}{
		"ActiveCount":         activeCount,
		"CompletedTodayCount": completedTodayCount,
		"TodayEarnings":       todayEarnings,
		"TotalEarnings":       totalEarnings,
		"IsTech":              true,
	}
}

// Helper for display
type JobViewModel struct {
	*core.Record
	DisplayTime string
}

func (h *TechHandler) JobsList(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.Redirect(http.StatusSeeOther, "/tech/login")
	}

	data := h.getTechCommonData(authRecord.Id)

	// Fetch jobs sorted by time (Earliest first)
	jobs, err := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && (job_status='assigned' || job_status='moving' || job_status='working')", authRecord.Id),
		"+booking_time",
		50,
		0,
		nil,
	)
	if err != nil {
		fmt.Printf("Error fetching jobs: %v\n", err)
	}

	// Prepare View Models
	var viewModels []JobViewModel
	for _, job := range jobs {
		vm := JobViewModel{Record: job}

		// Format Time
		rawTime := job.GetString("booking_time")
		parsedTime, err := time.Parse("2006-01-02 15:04", rawTime)
		if err != nil {
			parsedTime, _ = time.Parse("2006-01-02 15:04:05.000Z", rawTime)
		}

		if !parsedTime.IsZero() {
			// Duration: Ideally fetch from service. For now default 2h.
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

	data["Jobs"] = viewModels
	data["PageType"] = "tech_jobs"

	// Check for HTMX request for list partial
	if e.Request.Header.Get("HX-Target") == "job-list-container" {
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Note: partials/tech/jobs_list.html defines "tech/partials/jobs_list"
		return h.Templates.ExecuteTemplate(e.Response, "tech/partials/jobs_list", data)
	}

	// Use layout inheritance
	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/dashboard.html", data)
}

// 2. Cập nhật hàm chi tiết Job để lấy đầy đủ thông tin
func (h *TechHandler) JobDetail(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")

	// 1. Validate job exists
	job, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		// Job not found - render helpful 404 page
		return RenderPage(h.Templates, e, "layouts/tech.html", "tech/job_not_found.html", map[string]interface{}{
			"Message": "Công việc này không tồn tại hoặc đã bị xóa khỏi hệ thống.",
			"JobId":   jobID,
		})
	}

	// 2. Verify job belongs to this technician
	if job.GetString("technician_id") != e.Auth.Id {
		return RenderPage(h.Templates, e, "layouts/tech.html", "tech/job_not_found.html", map[string]interface{}{
			"Message": "Bạn không có quyền truy cập công việc này.",
			"JobId":   jobID,
		})
	}

	// 3. Get related data - Reports
	reports, _ := h.App.FindRecordsByFilter(
		"job_reports",
		fmt.Sprintf("booking_id='%s'", jobID),
		"-created", 1, 0, nil,
	)
	var report *core.Record
	if len(reports) > 0 {
		report = reports[0]
	}

	// 4. Get Invoice
	invoices, _ := h.App.FindRecordsByFilter(
		"invoices",
		fmt.Sprintf("booking_id='%s'", jobID),
		"-created", 1, 0, nil,
	)
	var invoice *core.Record
	if len(invoices) > 0 {
		invoice = invoices[0]
	}

	// 5. Calculate Progress
	progress := 0
	status := job.GetString("job_status")
	switch status {
	case "assigned":
		progress = 0
	case "moving":
		progress = 25
	case "working":
		progress = 50
	case "completed":
		progress = 100
	}

	// [NEW] Fetch Settings (if not already in common data, but JobDetail doesn't use getTechCommonData yet)
	// settings handled by middleware

	data := map[string]interface{}{
		"Job":             job,
		"Report":          report,  // Dữ liệu báo cáo (Ảnh sau)
		"Invoice":         invoice, // Dữ liệu hóa đơn (Tiền)
		"ProgressPercent": progress,
		"IsTech":          true,
	}

	// DEBUG
	fmt.Printf("🔍 JobDetail render - Job ID: %s, Status: %s, Progress: %d%%\n",
		job.Id, job.GetString("job_status"), progress)

	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/job_detail.html", data)
}

// ShowCompleteJob displays the completion form with inventory selection
// GET /tech/job/{id}/complete
func (h *TechHandler) ShowCompleteJob(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")

	job, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job không tồn tại")
	}

	// Get active inventory for selection
	inventory, _ := h.Inventory.GetActiveItems()

	// Get base service price
	serviceID := job.GetString("service_id")
	service, _ := h.App.FindRecordById("services", serviceID)
	laborPrice := 0.0
	if service != nil {
		laborPrice = service.GetFloat("price")
	}

	// settings handled by middleware

	data := map[string]interface{}{
		"Booking":    job,
		"Inventory":  inventory,
		"LaborPrice": laborPrice,
		"IsTech":     true,
	}

	// Use layout inheritance - Form specific (V2 with parts and invoice recalculation)
	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/forms/complete.html", data)
}

// SubmitCompleteJob processes the completion form with parts and invoice recalculation
func (h *TechHandler) SubmitCompleteJob(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	_, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job không tồn tại")
	}

	// 1. Save Photo Evidence
	files, _ := e.FindUploadedFiles("after_images")
	if len(files) == 0 {
		return e.String(400, "Bắt buộc phải có ảnh nghiệm thu")
	}

	// Create Job Report
	jobReports, _ := h.App.FindCollectionByNameOrId("job_reports")
	report := core.NewRecord(jobReports)
	report.Set("booking_id", jobID)
	report.Set("tech_id", e.Auth.Id)
	report.Set("photo_notes", e.Request.FormValue("notes"))

	// Handle file upload
	fileSlice := make([]any, len(files))
	for i, f := range files {
		fileSlice[i] = f
	}
	report.Set("after_images", fileSlice)

	if err := h.App.Save(report); err != nil {
		return e.String(500, "Lỗi lưu báo cáo: "+err.Error())
	}

	// 2. Parse and Process Parts Usage
	partsJSON := e.Request.FormValue("parts_json")
	var jobParts []services.JobPart

	if partsJSON != "" {
		err := json.Unmarshal([]byte(partsJSON), &jobParts)
		if err != nil {
			fmt.Printf("Error parsing parts_json: %v\n", err)
			return e.String(400, "Lỗi phân tích dữ liệu vật tư: "+err.Error())
		}

		// Record parts usage and deduct inventory
		totalPartsCost, err := h.Inventory.RecordPartsUsage(report.Id, jobParts)
		if err != nil {
			fmt.Printf("Error recording parts usage: %v\n", err)
			return e.String(400, "Lỗi xử lý vật tư: "+err.Error())
		}
		fmt.Printf("Parts recorded: %d items, total cost: %.2f đ\n", len(jobParts), totalPartsCost)
	}

	// 3. Generate/Recalculate Invoice
	invoice, err := h.InvoiceService.GenerateInvoice(jobID)
	if err != nil {
		fmt.Printf("Invoice generation error: %v\n", err)
		return e.String(500, "Lỗi tạo hóa đơn: "+err.Error()+". Vui lòng thử lại hoặc liên hệ Admin.")
	}

	// Ensure public hash for sharing
	if invoice.GetString("public_hash") == "" {
		invoice.Set("public_hash", fmt.Sprintf("%x", time.Now().UnixNano()))
		h.App.Save(invoice)
	}

	// 4. Update Booking Status to completed - MOVED TO PAYMENT
	// job.Set("job_status", "completed")
	// if err := h.App.Save(job); err != nil {
	// 	return e.String(500, "Lỗi cập nhật trạng thái job")
	// }

	// 5. Publish Events - MOVED TO PAYMENT
	// h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
	// 	Type:      "job.completed",
	// 	Timestamp: time.Now().Unix(),
	// 	Data: map[string]interface{}{
	// 		"booking_id":  jobID,
	// 		"tech_id":     e.Auth.Id,
	// 		"parts_count": len(jobParts),
	// 		"report_id":   report.Id,
	// 	},
	// })

	// Redirect to Invoice & Payment page instead of jobs list
	return e.Redirect(http.StatusSeeOther, fmt.Sprintf("/tech/job/%s/invoice-payment", jobID))
}

// ShowInvoicePayment displays the invoice, signature canvas, and payment options
func (h *TechHandler) ShowInvoicePayment(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")

	job, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job not found")
	}

	// Get or Generate Invoice
	// TÌM HOẶC TẠO HÓA ĐƠN
	invoices, _ := h.App.FindRecordsByFilter("invoices", fmt.Sprintf("booking_id='%s'", jobID), "", 1, 0, nil)
	var invoice *core.Record

	if len(invoices) > 0 {
		invoice = invoices[0]
	} else {
		// Thử tạo mới
		var errGen error
		invoice, errGen = h.InvoiceService.GenerateInvoice(jobID)

		// [FIX] NẾU LỖI -> ĐỪNG RENDER TRANG THANH TOÁN
		if errGen != nil || invoice == nil {
			// Log lỗi để debug
			fmt.Printf("Cannot generate invoice for job %s: %v\n", jobID, errGen)
			// Redirect về trang nghiệm thu để thợ làm lại báo cáo
			return e.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/tech/job/%s/complete", jobID))
		}
	}

	data := map[string]interface{}{
		"Job":     job,
		"Invoice": invoice,
		"IsTech":  true,
	}

	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/invoice_payment.html", data)
}

// --- Stub methods for existing routes ---

// 1. Hàm Dashboard: Phải gọi getTechCommonData
func (h *TechHandler) Dashboard(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.Redirect(302, "/tech/login")
	}

	// Lấy dữ liệu tổng quan (Số lượng, Doanh thu...)
	data := h.getTechCommonData(authRecord.Id)

	// DEBUG: Log actual auth record state
	fmt.Printf("🔍 Dashboard render - Tech ID: %s, active from e.Auth: %v\n", authRecord.Id, authRecord.GetBool("active"))

	// Đảm bảo truyền Jobs rỗng hoặc list mặc định để tránh lỗi nil pointer trong view
	// (Phần list job sẽ được HTMX load sau hoặc load ngay tùy ý)
	// Ở đây ta để HTMX load list job sau (lazy load) hoặc load luôn 5 job đầu tiên:
	// Load jobs (Earliest first)
	jobs, _ := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && (job_status='assigned' || job_status='moving' || job_status='working')", authRecord.Id),
		"+booking_time", 5, 0, nil,
	)

	var viewModels []JobViewModel
	for _, job := range jobs {
		vm := JobViewModel{Record: job}
		// Basic format for dashboard widget
		rawTime := job.GetString("booking_time")
		parsedTime, err := time.Parse("2006-01-02 15:04", rawTime)
		if err != nil {
			parsedTime, _ = time.Parse("2006-01-02 15:04:05.000Z", rawTime)
		}
		if !parsedTime.IsZero() {
			endTime := parsedTime.Add(2 * time.Hour)
			vm.DisplayTime = fmt.Sprintf("%02d:%02d - %02d:%02d",
				parsedTime.Hour(), parsedTime.Minute(),
				endTime.Hour(), endTime.Minute(),
			)
		}
		viewModels = append(viewModels, vm)
	}

	data["Jobs"] = viewModels

	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/dashboard.html", data)
}

func (h *TechHandler) ShowHistory(e *core.RequestEvent) error {
	data := h.getTechCommonData(e.Auth.Id)
	data["PageType"] = "history" // Corrected page type for nav highlighting
	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/history.html", data)
}

func (h *TechHandler) ShowProfile(e *core.RequestEvent) error {
	data := h.getTechCommonData(e.Auth.Id)
	data["TechName"] = e.Auth.Get("name")
	data["TechEmail"] = e.Auth.Email()
	data["PageType"] = "profile" // Corrected page type for nav highlighting
	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/profile.html", data)
}

func (h *TechHandler) ShowQuote(e *core.RequestEvent) error {
	return e.String(200, "Quote form")
}

func (h *TechHandler) SubmitQuote(e *core.RequestEvent) error {
	return e.String(200, "Quote submitted")
}

func (h *TechHandler) ShowReport(e *core.RequestEvent) error {
	return e.String(200, "Report form")
}

func (h *TechHandler) SubmitReport(e *core.RequestEvent) error {
	return e.String(200, "Report submitted")
}

// TechStream provides SSE endpoint for tech real-time notifications
// Each tech only receives events for their assigned jobs
func (h *TechHandler) TechStream(e *core.RequestEvent) error {
	// Verify tech authentication
	authRecord := e.Auth
	if authRecord == nil {
		return e.String(401, "Unauthorized")
	}

	// Extract tech_id from auth record
	techID := authRecord.Id

	// Set SSE headers
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")

	// Subscribe to tech-specific channel
	eventChan := h.Broker.Subscribe(broker.ChannelTech, techID)
	defer h.Broker.Unsubscribe(broker.ChannelTech, techID, eventChan)

	// Send initial connection event
	initialEvent := broker.Event{
		Type:      "connection.established",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"role":    "tech",
			"tech_id": techID,
		},
	}

	eventJSON, _ := json.Marshal(initialEvent)
	fmt.Fprintf(e.Response, "data: %s\n\n", eventJSON)
	e.Response.(http.Flusher).Flush()

	// Stream events (only for this tech)
	for {
		select {
		case event := <-eventChan:
			eventJSON, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(e.Response, "data: %s\n\n", eventJSON)
			e.Response.(http.Flusher).Flush()

		case <-e.Request.Context().Done():
			// Client disconnected
			return nil
		}
	}
}

// UpdateLocation updates the technician's current location
// POST /tech/location
func (h *TechHandler) UpdateLocation(e *core.RequestEvent) error {
	// Verify tech authentication
	authRecord := e.Auth
	if authRecord == nil {
		return e.String(401, "Unauthorized")
	}

	lat := e.Request.FormValue("lat")
	long := e.Request.FormValue("long")

	if lat == "" || long == "" {
		return e.String(400, "Missing coordinates")
	}

	authRecord.Set("current_lat", lat)
	authRecord.Set("current_long", long)
	authRecord.Set("location_updated_at", time.Now())

	if err := h.App.Save(authRecord); err != nil {
		return e.String(500, "Failed to update location")
	}

	return e.JSON(200, map[string]string{"status": "ok"})
}

// HandleTechCheckIn processes the technician's check-in request
// POST /api/tech/bookings/{id}/checkin
func (h *TechHandler) HandleTechCheckIn(e *core.RequestEvent) error {
	bookingID := e.Request.PathValue("id")
	latStr := e.Request.FormValue("lat")
	longStr := e.Request.FormValue("long")

	if latStr == "" || longStr == "" {
		return e.JSON(400, map[string]string{"error": "Missing coordinates"})
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		return e.JSON(400, map[string]string{"error": "Invalid latitude"})
	}

	long, err := strconv.ParseFloat(longStr, 64)
	if err != nil {
		return e.JSON(400, map[string]string{"error": "Invalid longitude"})
	}

	// Call Service
	err = h.BookingService.TechCheckIn(bookingID, lat, long)
	if err != nil {
		// Return friendly error message
		return e.JSON(400, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]string{
		"message": "Check-in thành công",
		"status":  "arrived", // Return new status for Frontend to update
	})
}

// POST /tech/job/{id}/evidence
func (h *TechHandler) UploadEvidence(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	evidenceType := e.Request.FormValue("type") // "before" or "after"

	// 1. Tìm hoặc tạo Job Report
	reports, _ := h.App.FindRecordsByFilter("job_reports", fmt.Sprintf("booking_id='%s'", jobID), "-created", 1, 0, nil)

	var report *core.Record
	jobReports, _ := h.App.FindCollectionByNameOrId("job_reports")

	if len(reports) > 0 {
		report = reports[0]
	} else {
		// Nếu chưa có report thì tạo mới
		report = core.NewRecord(jobReports)
		report.Set("booking_id", jobID)
		report.Set("tech_id", e.Auth.Id)
	}

	// 2. Xử lý file upload
	files, _ := e.FindUploadedFiles("images")
	if len(files) == 0 {
		return e.String(400, "Vui lòng chọn ảnh")
	}

	// Chuyển đổi file sang slice interface{} để lưu vào PocketBase
	fileSlice := make([]any, len(files))
	for i, f := range files {
		fileSlice[i] = f
	}

	// 3. Cập nhật trường tương ứng (cộng dồn ảnh nếu cần, ở đây ta lưu đè hoặc thêm mới tuỳ logic PB)
	// Lưu ý: PocketBase mặc định sẽ append nếu dùng API, nhưng với core app thì set lại slice.
	// Để đơn giản cho MVP, ta cho phép upload nhiều lần, mỗi lần lưu vào field tương ứng.
	fieldName := "before_images"
	if evidenceType == "after" {
		fieldName = "after_images"
	}

	// Lưu file vào record
	report.Set(fieldName, fileSlice)

	if err := h.App.Save(report); err != nil {
		return e.String(500, "Lỗi lưu ảnh: "+err.Error())
	}

	// Trả về thành công (hoặc HTML partial nếu dùng HTMX để hiển thị lại ảnh vừa up)
	return e.String(200, "Đã lưu ảnh thành công")
}
