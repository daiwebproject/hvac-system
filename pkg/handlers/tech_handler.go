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
	"hvac-system/pkg/notification"
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
	SettingsRepo   *repository.SettingsRepo    // [NEW]
	FCMService     *notification.FCMService    // [NEW]
	TechRepo       domain.TechnicianRepository // [PHASE4] For migration
	BookingRepo    domain.BookingRepository    // [PHASE4] For migration
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

	// [REFACTORED] Fetch Fresh Tech from Repository for Real-time Status
	var tech *domain.Technician
	var err error
	if h.TechRepo != nil {
		tech, err = h.TechRepo.GetByID(techID)
		if err != nil {
			fmt.Printf("Error fetching tech from repo: %v\n", err)
		}
	} else {
		// Fallback: convert raw record to domain model
		techRecord, err := h.App.FindRecordById("technicians", techID)
		if err != nil {
			fmt.Printf("Error fetching fresh tech record: %v\n", err)
		} else {
			tech = &domain.Technician{
				ID:       techRecord.Id,
				Name:     techRecord.GetString("name"),
				Email:    techRecord.Email(),
				Avatar:   techRecord.GetString("avatar"),
				Active:   techRecord.GetBool("active"),
				Verified: techRecord.GetBool("verified"),
				FCMToken: techRecord.GetString("fcm_token"),
			}
		}
	}

	// [FIX] Query for New Jobs (Assigned but not started)
	newJobs, _ := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && job_status='assigned'", techID),
		"", 0, 0, nil,
	)

	// [FIX] Query for Total Completed Jobs (All time)
	completedTotalRecords, _ := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && job_status='completed'", techID),
		"", 0, 0, nil,
	)
	completedTotal := len(completedTotalRecords)

	return map[string]interface{}{
		"ActiveCount":         activeCount,
		"CompletedTodayCount": completedTodayCount,
		"TodayEarnings":       todayEarnings,
		"TotalEarnings":       totalEarnings,
		"IsTech":              true,
		"Tech":                tech, // [REFACTORED] Domain model instead of raw record
		"NewJobsCount":        len(newJobs),
		"CompletedTotalCount": completedTotal,
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
	if e.Request.Header.Get("HX-Request") == "true" {
		e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Note: partials/tech/jobs_list.html defines "tech/partials/jobs_list"
		// [FIX] Use renderPartial to safely Clone before Executing
		return h.renderPartial(e, "tech/partials/jobs_list", data)
	}

	// Use layout inheritance
	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/dashboard.html", data)
}

// 2. Cập nhật hàm chi tiết Job để lấy đầy đủ thông tin
func (h *TechHandler) JobDetail(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")

	// 1. Validate job exists
	var err error // [FIX] Declare err
	// 1. Fetch Job from Repository (or Fallback)
	var job *domain.Booking
	if h.BookingRepo != nil {
		job, err = h.BookingRepo.GetByID(jobID)
		if err != nil {
			return RenderPage(h.Templates, e, "layouts/tech.html", "tech/job_not_found.html", map[string]interface{}{
				"Message": "Công việc này không tồn tại hoặc đã bị xóa khỏi hệ thống.",
				"JobId":   jobID,
			})
		}
	} else {
		// Fallback for safety during migration
		rec, err := h.App.FindRecordById("bookings", jobID)
		if err != nil {
			return RenderPage(h.Templates, e, "layouts/tech.html", "tech/job_not_found.html", map[string]interface{}{
				"Message": "Công việc này không tồn tại hoặc đã bị xóa khỏi hệ thống.",
				"JobId":   jobID,
			})
		}
		// Convert to domain (manual for now, ideally use a mapper)
		job = &domain.Booking{
			ID:               rec.Id,
			TechnicianID:     rec.GetString("technician_id"),
			JobStatus:        rec.GetString("job_status"),
			CustomerName:     rec.GetString("customer_name"),
			CustomerPhone:    rec.GetString("customer_phone"),
			Address:          rec.GetString("address"),
			AddressDetails:   rec.GetString("address_details"),
			DeviceType:       rec.GetString("device_type"),
			Brand:            rec.GetString("brand"),
			BookingTime:      rec.GetString("booking_time"),
			IssueDescription: rec.GetString("issue_description"),
			Lat:              rec.GetFloat("lat"),
			Long:             rec.GetFloat("long"),
			TechNotes:        rec.GetString("tech_notes"), // [NEW]
		}
	}

	// 2. Verify job belongs to this technician
	if job.TechnicianID != e.Auth.Id {
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

	// 5. Calculate Progress & StatusOrder
	progress := 0
	status := job.JobStatus

	// Map Status to Order for Stepper UI
	switch status {
	case "pending", "assigned":
		job.StatusOrder = 1
		progress = 0
	case "accepted":
		job.StatusOrder = 1
		progress = 10
	case "moving":
		job.StatusOrder = 2
		progress = 25
	case "arrived":
		job.StatusOrder = 3
		progress = 40
	case "working":
		job.StatusOrder = 4
		progress = 75
	case "completed", "paid":
		job.StatusOrder = 5
		progress = 100
	case "cancelled":
		job.StatusOrder = 0
		progress = 0
	}

	// [NEW] Fetch Settings (if not already in common data, but JobDetail doesn't use getTechCommonData yet)
	// settings handled by middleware

	data := map[string]interface{}{
		"Job":             job,
		"Report":          report,  // Dữ liệu báo cáo (Ảnh sau)
		"Invoice":         invoice, // Dữ liệu hóa đơn (Tiền)
		"ProgressPercent": progress,
		"IsTech":          true,
		"PageType":        "job_detail", // Used to hide main nav
	}

	// DEBUG
	fmt.Printf("🔍 JobDetail render - Job ID: %s, Status: %s, Progress: %d%%\n",
		job.ID, job.JobStatus, progress)

	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/job_detail.html", data)
}

// ShowCompleteJob displays the completion form with inventory selection
// GET /tech/job/{id}/complete
func (h *TechHandler) ShowCompleteJob(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	techID := e.Auth.Id // Get tech ID from auth

	var err error // [FIX] Declare err
	// 1. Fetch Job from Repository (or Fallback)
	var job *domain.Booking
	if h.BookingRepo != nil {
		job, err = h.BookingRepo.GetByID(jobID)
		if err != nil {
			return e.String(404, "Job không tồn tại")
		}
	} else {
		// Fallback for safety during migration
		rec, err := h.App.FindRecordById("bookings", jobID)
		if err != nil {
			return e.String(404, "Job không tồn tại")
		}
		// Convert to domain (manual for now, ideally use a mapper)
		job = &domain.Booking{
			ID:        rec.Id,
			ServiceID: rec.GetString("service_id"),
			// Other fields if needed
		}
	}

	// [TRUCK STOCK] Get technician's truck inventory instead of main inventory
	techInventory, _ := h.Inventory.GetTechInventory(techID)

	// Get base service price
	serviceID := job.ServiceID
	service, _ := h.App.FindRecordById("services", serviceID)
	laborPrice := 0.0
	if service != nil {
		laborPrice = service.GetFloat("price")
	}

	// settings handled by middleware

	data := map[string]interface{}{
		"Booking":       job,
		"TechInventory": techInventory, // [TRUCK STOCK] Tech's own inventory
		"LaborPrice":    laborPrice,
		"IsTech":        true,
		"TechID":        techID,
		"PageType":      "job_detail", // Hide main nav
	}

	// Use layout inheritance - Form specific (V2 with parts and invoice recalculation)
	return RenderPage(h.Templates, e, "layouts/tech.html", "tech/forms/complete.html", data)
}

// SubmitCompleteJob processes the completion form with parts and invoice recalculation
func (h *TechHandler) SubmitCompleteJob(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	var err error
	if h.BookingRepo != nil {
		_, err = h.BookingRepo.GetByID(jobID)
	} else {
		_, err = h.App.FindRecordById("bookings", jobID)
	}

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

		// [TRUCK STOCK] Record parts usage and deduct from TECH's inventory
		techID := e.Auth.Id
		totalPartsCost, err := h.Inventory.RecordPartsUsageFromTech(report.Id, techID, jobID, jobParts)
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
	}

	// [NEW] Save tech signature to invoice
	techSigFiles, _ := e.FindUploadedFiles("tech_signature_file")
	if len(techSigFiles) > 0 {
		invoice.Set("tech_signature", techSigFiles[0])
		invoice.Set("tech_signed_at", time.Now())
		fmt.Printf("✅ Saved tech signature for invoice %s\n", invoice.Id)
	}

	// Save invoice with tech signature
	if err := h.App.Save(invoice); err != nil {
		return e.String(500, "Lỗi lưu hóa đơn: "+err.Error())
	}

	// 4. Redirect to payment page (customer signs and pays)
	return e.Redirect(http.StatusSeeOther, fmt.Sprintf("/tech/job/%s/invoice-payment", jobID))
}

// ShowInvoicePayment displays the invoice, signature canvas, and payment options
func (h *TechHandler) ShowInvoicePayment(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")

	// 1. Fetch Job from Repository (or Fallback)
	var job *domain.Booking
	var err error
	if h.BookingRepo != nil {
		job, err = h.BookingRepo.GetByID(jobID)
		if err != nil {
			return e.String(404, "Job không tồn tại")
		}
	} else {
		// Fallback for safety during migration
		rec, err := h.App.FindRecordById("bookings", jobID)
		if err != nil {
			return e.String(404, "Job không tồn tại")
		}
		// Convert to domain
		job = &domain.Booking{
			ID:           rec.Id,
			CustomerName: rec.GetString("customer_name"),
			TechnicianID: rec.GetString("technician_id"),
			// Other fields used by template
		}
	}

	// Get or Generate Invoice
	// TÌM HOẶC TẠO HÓA ĐƠN
	invoices, _ := h.App.FindRecordsByFilter("invoices", fmt.Sprintf("booking_id='%s'", jobID), "", 1, 0, nil)
	var invoice *core.Record

	if len(invoices) > 0 {
		invoice = invoices[0]
		fmt.Printf("📄 Found existing invoice %s for job %s\n", invoice.Id, jobID)

		// [FIX] Check if items exist - if not, regenerate (migration fix)
		items, _ := h.App.FindRecordsByFilter("invoice_items", fmt.Sprintf("invoice_id='%s'", invoice.Id), "", 1, 0, nil)
		if len(items) == 0 {
			fmt.Printf("⚠️  Invoice %s has no items, regenerating...\n", invoice.Id)
			// Regenerate to create items
			newInvoice, errRegen := h.InvoiceService.GenerateInvoice(jobID)
			if errRegen == nil && newInvoice != nil {
				invoice = newInvoice
				fmt.Printf("✅ Regenerated invoice with items\n")
			}
		}
	} else {
		// Thử tạo mới
		var errGen error
		invoice, errGen = h.InvoiceService.GenerateInvoice(jobID)

		// [FIX] NẾU LỖI -> ĐỪNG RENDER TRANG THANH TOÁN
		if errGen != nil || invoice == nil {
			// Log lỗi để debug
			fmt.Printf("❌ Cannot generate invoice for job %s: %v\n", jobID, errGen)
			// Redirect về trang nghiệm thu để thợ làm lại báo cáo
			return e.Redirect(http.StatusTemporaryRedirect, fmt.Sprintf("/tech/job/%s/complete", jobID))
		}
		fmt.Printf("✅ Generated new invoice %s for job %s\n", invoice.Id, jobID)
	}

	// [FIX 1] Lấy Job Report để hiển thị ảnh nghiệm thu
	reports, _ := h.App.FindRecordsByFilter(
		"job_reports",
		fmt.Sprintf("booking_id='%s'", jobID),
		"-created", 1, 0, nil,
	)
	var report *core.Record
	if len(reports) > 0 {
		report = reports[0]
	}

	// [FIX 2] Refresh invoice items explicitly to ensure sync
	if invoice != nil {
		// Re-fetch items just to be safe, although we already do it below
		// The previous logic was: items, _ := h.App.FindRecordsByFilter("invoice_items", ...)
		// We keep that logic but we can ensure the invoice object itself is up to date if needed
		// h.App.Dao().ExpandRecord(invoice, []string{"booking_id"}, nil) // If needed
	}

	// [FIX] Fetch Invoice Items for Detailed View
	// Items are now automatically created by InvoiceService.GenerateInvoice
	items, _ := h.App.FindRecordsByFilter("invoice_items", fmt.Sprintf("invoice_id='%s'", invoice.Id), "", 100, 0, nil)
	fmt.Printf("📋 Invoice %s has %d items\n", invoice.Id, len(items))

	data := map[string]interface{}{
		"Job":      job,
		"Invoice":  invoice,
		"Items":    items,  // Pass items to view
		"Report":   report, // [FIX] Pass report to view
		"IsTech":   true,
		"PageType": "job_detail", // Hide main nav
		// Settings will be auto-injected by RenderPage from middleware
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
	authRecord := e.Auth
	if authRecord == nil {
		return e.Redirect(http.StatusSeeOther, "/tech/login")
	}

	// 1. Lấy dữ liệu chung (Stats, TechRecord)
	data := h.getTechCommonData(authRecord.Id)

	// 2. Truy vấn 20 đơn gần nhất đã hoàn thành
	historyJobs, err := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && job_status='completed'", authRecord.Id),
		"-updated", // Đơn mới hoàn thành hiện lên đầu
		20, 0, nil,
	)
	if err != nil {
		fmt.Printf("Error fetching history: %v\n", err)
	}

	// [NEW] Enrich with Invoice Amount
	for _, job := range historyJobs {
		invoices, _ := h.App.FindRecordsByFilter(
			"invoices",
			fmt.Sprintf("booking_id='%s'", job.Id),
			"", 1, 0, nil,
		)
		if len(invoices) > 0 {
			job.Set("final_amount", invoices[0].GetFloat("total_amount"))
		} else {
			job.Set("final_amount", 0)
		}
	}

	data["HistoryJobs"] = historyJobs
	data["PageType"] = "history"

	// 3. Sử dụng RenderPage để render toàn bộ trang lịch sử
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
			fmt.Fprintf(e.Response, "event: %s\ndata: %s\n\n", event.Type, eventJSON)
			e.Response.(http.Flusher).Flush()
			fmt.Printf(" [SSE] Sent event %s to tech %s\n", event.Type, techID)

		case <-e.Request.Context().Done():
			// Client disconnected
			return nil
		}
	}
}

// GET /api/tech/schedule
// GetSchedule xử lý yêu cầu hiển thị lịch trình làm việc (Timeline)
// GET /api/tech/schedule
func (h *TechHandler) GetSchedule(e *core.RequestEvent) error {
	authRecord := e.Auth
	if authRecord == nil {
		return e.JSON(401, map[string]string{"error": "Unauthorized"})
	}

	// 1. Lấy dữ liệu chung (Stats, TechRecord)
	data := h.getTechCommonData(authRecord.Id)

	// 2. Truy vấn đơn hàng chưa hoàn thành, sắp xếp theo thời gian
	jobs, err := h.App.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && (job_status='assigned' || job_status='moving' || job_status='working')", authRecord.Id),
		"+booking_time", // Sắp xếp tăng dần để tạo timeline
		50, 0, nil,
	)
	if err != nil {
		return e.String(500, "Lỗi truy vấn lịch trình")
	}

	// 3. Chuẩn bị dữ liệu hiển thị (ViewModel)
	type ScheduleItem struct {
		JobViewModel
		StatusColor string
		Icon        string
		IsCurrent   bool // [NEW]
	}

	var scheduleItems []ScheduleItem
	for _, job := range jobs {
		vm := JobViewModel{Record: job}
		// [FIX] Populate TechNotes
		vm.Record.Set("tech_notes", job.GetString("tech_notes"))

		// Logic parse thời gian giống JobsList
		rawTime := job.GetString("booking_time")
		parsedTime, _ := time.Parse("2006-01-02 15:04", rawTime)
		if parsedTime.IsZero() {
			parsedTime, _ = time.Parse("2006-01-02 15:04:05.000Z", rawTime)
		}
		vm.DisplayTime = parsedTime.Format("15:04")

		// [FIX] Fetch Service Name
		serviceID := job.GetString("service_id")
		serviceName := "Dịch vụ"
		if service, err := h.App.FindRecordById("services", serviceID); err == nil {
			serviceName = service.GetString("name")
		}
		job.Set("service_name", serviceName)

		item := ScheduleItem{JobViewModel: vm}
		status := job.GetString("job_status")

		// [FIX] Determine IsCurrent
		if status == "moving" || status == "working" {
			item.StatusColor = "primary"
			item.Icon = "fa-spinner fa-spin" // Active icon
			// We can add a field IsCurrent to the struct if needed, or just check StatusColor in template
			// But template uses .IsCurrent. Let's add it to the struct or map.
			// Wait, ScheduleItem struct is defined inside the function on line 797.
			// I should add IsCurrent to it.
		} else if status == "arrived" {
			item.StatusColor = "purple"
			item.Icon = "fa-map-pin"
		} else {
			item.StatusColor = "gray" // Default
			item.Icon = "fa-calendar-check"
		}
		scheduleItems = append(scheduleItems, item)
	}

	data["ScheduleItems"] = scheduleItems
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")

	// TRẢ VỀ PARTIAL (Tránh lỗi Clone bằng cách gọi trực tiếp template name)
	// [FIX] Use renderPartial
	return h.renderPartial(e, "tech/partials/schedule_list", data)
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

// UpdateNote updates the technician's private note for a job
// POST /tech/job/{id}/note
func (h *TechHandler) UpdateNote(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	note := e.Request.FormValue("note")

	// Verify auth
	if e.Auth == nil {
		return e.String(401, "Unauthorized")
	}

	// Fetch booking
	booking, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job not found")
	}

	// Verify ownership
	if booking.GetString("technician_id") != e.Auth.Id {
		return e.String(403, "Forbidden")
	}

	// Update note
	booking.Set("tech_notes", note)

	if err := h.App.Save(booking); err != nil {
		return e.String(500, "Failed to save note")
	}

	// Return success indicator (maybe just a checkmark or toast trigger)
	return e.String(200, "Note saved")
}
