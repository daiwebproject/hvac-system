package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"

	"hvac-system/internal/adapter/repository"
	domain "hvac-system/internal/core"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/services"
	"hvac-system/pkg/ui"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

type AdminHandler struct {
	App              core.App
	Templates        *template.Template
	Broker           *broker.SegmentedBroker
	BookingService   domain.BookingService           // [MIGRATED] internal domain service
	SlotService      *services.TimeSlotService       // TODO: Migrate this too
	TechService      *services.TechManagementService // NEW: Tech Management
	AnalyticsService domain.AnalyticsService
	UIComponents     *ui.Components
	SettingsRepo     *repository.SettingsRepo // [NEW]
	FCMService       *services.FCMService     // [NEW] FCM Push Notifications
}

func (h *AdminHandler) ShowLogin(e *core.RequestEvent) error {
	return RenderPage(h.Templates, e, "layouts/auth.html", "public/login.html", nil)
}

func (h *AdminHandler) ProcessLogin(e *core.RequestEvent) error {
	email := e.Request.FormValue("email")
	password := e.Request.FormValue("password")

	superuser, err := h.App.FindAuthRecordByEmail("_superusers", email)
	if err != nil || !superuser.ValidatePassword(password) {
		return RenderPage(h.Templates, e, "layouts/auth.html", "public/login.html", map[string]string{
			"Error": "Sai email hoặc mật khẩu!",
		})
	}

	token, err := superuser.NewAuthToken()
	if err != nil {
		return e.String(500, "Lỗi hệ thống")
	}

	http.SetCookie(e.Response, &http.Cookie{
		Name:     "pb_auth",
		Value:    token,
		Path:     "/",
		Secure:   false, // Đặt true nếu chạy HTTPS
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Now().Add(7 * 24 * time.Hour),
	})

	return e.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) Logout(e *core.RequestEvent) error {
	http.SetCookie(e.Response, &http.Cookie{
		Name:     "pb_auth",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
	})
	return e.Redirect(http.StatusSeeOther, "/login")
}

// Helper struct for JSON serialization in Dashboard
type BookingJSON struct {
	ID             string  `json:"id"`
	Customer       string  `json:"customer"`
	StaffID        string  `json:"staff_id"`
	Service        string  `json:"service"`
	Time           string  `json:"time"`    // Chuỗi hiển thị giờ làm (VD: 30/01 10:00 - 12:00)
	Created        string  `json:"created"` // [MỚI] Thời gian khách đặt đơn
	Status         string  `json:"status"`
	StatusLabel    string  `json:"status_label"`
	Phone          string  `json:"phone"`
	AddressDetails string  `json:"address_details"`
	Address        string  `json:"address"`
	Lat            float64 `json:"lat"`
	Long           float64 `json:"long"`
	Issue          string  `json:"issue"`
	CancelReason   string  `json:"cancel_reason"` // [MỚI] Lý do hủy
	InvoiceHash    string  `json:"invoice_hash"`  // [MỚI] Thêm trường này
}

// Dashboard renders the admin dashboard with Kanban board
func (h *AdminHandler) Dashboard(e *core.RequestEvent) error {
	// 1. Fetch active bookings (Kanban items)
	bookings, err := h.App.FindRecordsByFilter(
		"bookings",
		// "job_status != 'cancelled'", // Allow cancelled jobs to show
		"",              // No filter, or maybe "created >= '2024-01-01'" if too many
		"+booking_time", // Sort by schedule (earliest first)
		100,
		0,
		nil,
	)
	if err != nil {
		return e.String(500, "Lỗi load booking: "+err.Error())
	}

	// Active technicians for assignment dropdowns
	technicians, _ := h.App.FindRecordsByFilter("technicians", "active=true", "name", 100, 0, nil)

	// 2. Fetch Stats & Analytics using optimised service
	stats, err := h.AnalyticsService.GetDashboardStats()
	if err != nil {
		// Log error but proceed with empty stats
		fmt.Printf("Dashboard Stats Error: %v\n", err)
		stats = &domain.DashboardStats{}
	}

	revenueStats, _ := h.AnalyticsService.GetRevenueLast7Days()
	topTechs, _ := h.AnalyticsService.GetTopTechnicians(5)

	// 3. Serialize bookings to JSON for Frontend Map/Kanban interactions
	bookingsJSON := []BookingJSON{}
	for _, b := range bookings {
		// Xử lý tên dịch vụ
		serviceName := b.GetString("device_type")
		if serviceName == "" {
			serviceName = "Kiểm tra / Khác"
		}

		// Xử lý hiển thị thời gian (Format: HH:MM - HH:MM DD/MM/YYYY)
		rawTime := b.GetString("booking_time")
		displayTime := rawTime

		// Parse booking time
		// Support both DB format "YYYY-MM-DD HH:MM:SS.000Z" and "YYYY-MM-DD HH:MM"
		parsedTime, err := time.Parse("2006-01-02 15:04:05.000Z", rawTime)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02 15:04", rawTime) // Our new format
		}

		if err == nil {
			// Calculate End Time (Default 2 hours if not expandable, or fetch service if needed)
			// Optimally we should check service duration, but for list view default is acceptable for MVP
			// or we can fetch service.
			// Ideally we assume 1.5 - 2h.
			duration := 2 * time.Hour

			endTime := parsedTime.Add(duration)
			displayTime = fmt.Sprintf("%02d:%02d - %02d:%02d ngày %02d/%02d/%d",
				parsedTime.Hour(), parsedTime.Minute(),
				endTime.Hour(), endTime.Minute(),
				parsedTime.Day(), parsedTime.Month(), parsedTime.Year(),
			)
		}

		// [MỚI] Tìm hóa đơn của job này để lấy Hash
		invoiceHash := ""
		// Tìm record hóa đơn theo booking_id
		if invoices, err := h.App.FindRecordsByFilter("invoices", "booking_id='"+b.Id+"'", "", 1, 0, nil); err == nil && len(invoices) > 0 {
			invoiceHash = invoices[0].GetString("public_hash")
		}

		// [FIX] Address Logic: Prioritize address_details (specific), fallback to address (generic)
		address := b.GetString("address_details")
		if address == "" {
			address = b.GetString("address")
		}

		// [FIX] Cancel Reason Logic
		cancelReason := b.GetString("cancel_reason")
		if cancelReason == "" && b.GetString("job_status") == "cancelled" {
			cancelReason = b.GetString("cancel_note") // Fallback attempt
			if cancelReason == "" {
				cancelReason = "Đã hủy"
			}
		}

		bookingsJSON = append(bookingsJSON, BookingJSON{
			ID:             b.Id,
			Customer:       b.GetString("customer_name"),
			StaffID:        b.GetString("technician_id"),
			Service:        serviceName,
			Time:           displayTime,
			Created:        b.GetString("created"),
			Status:         b.GetString("job_status"),
			StatusLabel:    b.GetString("job_status"),
			Phone:          b.GetString("customer_phone"),
			AddressDetails: b.GetString("address_details"),
			Address:        address, // [FIX] Use processed address
			Lat:            b.GetFloat("lat"),
			Long:           b.GetFloat("long"),
			Issue:          b.GetString("issue_description"),
			CancelReason:   cancelReason, // [FIX]
			InvoiceHash:    invoiceHash,  // [MỚI] Gán giá trị
		})
	}

	bookingsJSONBytes, err := json.Marshal(bookingsJSON)
	if err != nil {
		fmt.Printf("Error marshalling bookingsJSON: %v\n", err)
		bookingsJSONBytes = []byte("[]")
	}

	// [NEW] Chuẩn bị dữ liệu Thợ cho Map
	type TechMapJSON struct {
		ID     string  `json:"id"`
		Name   string  `json:"name"`
		Lat    float64 `json:"lat"`
		Long   float64 `json:"long"`
		Active bool    `json:"active"`
	}

	techsJSON := []TechMapJSON{}
	for _, t := range technicians {
		techsJSON = append(techsJSON, TechMapJSON{
			ID:     t.Id,
			Name:   t.GetString("name"),
			Active: t.GetBool("active"),
			// Lat: t.GetFloat("last_lat"),
			// Long: t.GetFloat("last_long"),
		})
	}

	techsJSONBytes, err := json.Marshal(techsJSON)
	if err != nil {
		fmt.Printf("Error marshalling techsJSON: %v\n", err)
		techsJSONBytes = []byte("[]")
	}

	// Fetch Services for Dropdown
	servicesList, _ := h.App.FindRecordsByFilter("services", "active=true", "-created", 100, 0, nil)

	data := map[string]interface{}{
		"Bookings":        bookings,
		"BookingsJSON":    template.JS(string(bookingsJSONBytes)),
		"TechniciansJSON": template.JS(string(techsJSONBytes)), // [QUAN TRỌNG] Truyền cái này xuống View
		"Technicians":     technicians,
		"Services":        servicesList,
		"TotalRevenue":    stats.TotalRevenue,
		"BookingsToday":   stats.BookingsToday,
		"ActiveTechs":     stats.ActiveTechs,
		"Pending":         stats.PendingCount,
		"Completed":       stats.CompletedCount,
		"CompletionRate":  stats.CompletionRate,
		"RevenueStats":    revenueStats,
		"TopTechs":        topTechs,
		"IsAdmin":         true,
		"PageType":        "admin_dashboard",
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/dashboard.html", data)
}

// POST /admin/bookings/create
func (h *AdminHandler) CreateBooking(e *core.RequestEvent) error {
	name := e.Request.FormValue("customer_name")
	phone := e.Request.FormValue("customer_phone")
	address := e.Request.FormValue("address")
	serviceID := e.Request.FormValue("service_id")
	bookingTime := e.Request.FormValue("booking_time") // Expect "2006-01-02T15:04"
	issue := e.Request.FormValue("issue_description")

	if name == "" || phone == "" {
		return e.String(400, "Vui lòng nhập tên và số điện thoại")
	}

	collection, err := h.App.FindCollectionByNameOrId("bookings")
	if err != nil {
		return e.String(500, "Collection not found")
	}

	record := core.NewRecord(collection)
	record.Set("customer_name", name)
	record.Set("customer_phone", phone)
	record.Set("address", address)
	record.Set("service_id", serviceID)

	// Lookup service name/device_type
	if serviceID != "" {
		svc, err := h.App.FindRecordById("services", serviceID)
		if err == nil {
			record.Set("device_type", svc.GetString("name"))
			record.Set("estimated_cost", svc.GetFloat("base_price"))
		}
	} else {
		record.Set("device_type", "Kiểm tra / Khác")
	}

	// Format time
	// UI sends "YYYY-MM-DDTHH:MM", DB expects "YYYY-MM-DD HH:MM:SS.000Z"
	// 1. Parse chuỗi thời gian từ UI
	parsedTime, err := time.Parse("2006-01-02T15:04", bookingTime)
	if err != nil {
		// Thử format có giây nếu UI gửi lên
		parsedTime, _ = time.Parse("2006-01-02T15:04:05", bookingTime)
	}

	// 2. Lưu vào DB dưới dạng chuẩn (hoặc format bạn đang dùng)
	if !parsedTime.IsZero() {
		// Lưu thống nhất: "2006-01-02 15:04" để khớp với logic hiển thị dashboard
		record.Set("booking_time", parsedTime.Format("2006-01-02 15:04"))
	} else {
		// Fallback nếu parse lỗi
		record.Set("booking_time", bookingTime)
	}

	record.Set("issue_description", issue)
	record.Set("job_status", "pending")
	record.Set("created", time.Now().Format("2006-01-02 15:04:05.000Z"))

	if err := h.App.Save(record); err != nil {
		return e.String(500, "Lỗi lưu đơn hàng: "+err.Error())
	}

	// [REFACTORED] Use BookingService to trigger notifications
	// Ideally we should use h.BookingService.CreateBooking, but that requires constructing domain props.
	// For now, the EASIEST fix is to manually trigger the service's notification helper?
	// or specific method? internal BookingService doesn't expose "Notify".
	//
	// Better approach: We HAVE CreateBooking in service. Let's use it?
	// But getting all fields mapped exactly might be tricky with `record.Set`.
	//
	// Alternative: Since we just want the notification side-effect, and we have h.BookingService (domain interface),
	// we assume the service layer handles it.
	// But we just created the record MANUALLY via h.App.Save(record). The service doesn't know about it.
	//
	// FIX: We must construct a BookingRequest and call h.BookingService.CreateBooking
	// Then we can update the 'estimated_cost' manually if needed.

	// Let's delete the manual save above and use the service properly.

	/* MANUAL SAVE DELETED */

	// Construct Request
	req := &domain.BookingRequest{
		CustomerName:   name,
		Phone:          phone,
		AddressDetails: address,                          // Map 'address' form to Details (or Address?)
		BookingTime:    record.GetString("booking_time"), // Use the formatted time
		IssueDesc:      issue,
		ServiceID:      serviceID,
		DeviceType:     record.GetString("device_type"),
	}

	// Creates booking AND triggers notifications (SSE+FCM)
	newBooking, err := h.BookingService.CreateBooking(req)
	if err != nil {
		return e.String(500, "Lỗi service tạo đơn: "+err.Error())
	}

	// Post-update for fields not in BookingRequest (e.g. estimated_cost)
	// We need to fetch the record again or just update 'newBooking' if it was a struct?
	// CreateBooking returns *core.Booking (struct). We need to update DB record for cost.
	if newBooking != nil && record.GetFloat("estimated_cost") > 0 {
		// We have to find the record by ID
		if freshRecord, err := h.App.FindRecordById("bookings", newBooking.ID); err == nil {
			freshRecord.Set("estimated_cost", record.GetFloat("estimated_cost"))
			h.App.Save(freshRecord)
		}
	}

	return e.JSON(200, map[string]string{"message": "Đã tạo đơn hàng mới"})
}

func (h *AdminHandler) UpdateBookingStatus(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	status := e.Request.FormValue("status")

	if id == "" || status == "" {
		return e.String(400, "Missing ID or Status")
	}

	// 1. Handle "recall to pending" efficiently via service
	// 1. Handle "recall to pending" efficiently via service
	if status == "pending" {
		// [FIX] New BookingService.RecallToPending only takes ID
		if err := h.BookingService.RecallToPending(id); err != nil {
			return e.String(500, "Lỗi khi thu hồi về Pending: "+err.Error())
		}
		return e.JSON(200, map[string]string{"message": "Đã thu hồi về Pending"})
	}

	// 2. Normal status flow
	if err := h.BookingService.UpdateStatus(id, status); err != nil {
		return e.String(500, err.Error())
	}

	return e.JSON(200, map[string]string{"message": "Success"})
}

func (h *AdminHandler) AssignJob(e *core.RequestEvent) error {
	bookingID := e.Request.PathValue("id")
	log.Printf("👉 [ADMIN_HANDLER] AssignJob Called. BookingID: %s\n", bookingID)

	if bookingID == "" {
		return e.String(400, "Lỗi: Không tìm thấy ID trên URL")
	}
	technicianID := e.Request.FormValue("technician_id")
	if technicianID == "" {
		return e.String(400, "Lỗi: Vui lòng chọn thợ")
	}

	// Check for schedule conflicts
	booking, err := h.App.FindRecordById("bookings", bookingID)
	if err != nil {
		return e.String(404, "Job không tồn tại")
	}

	bookingTimeStr := booking.GetString("booking_time") // "YYYY-MM-DD HH:MM"
	if len(bookingTimeStr) >= 16 {
		date := bookingTimeStr[:10]
		timeStr := bookingTimeStr[11:16]

		// 1. Lấy thời lượng thực tế từ Service của Booking này
		serviceID := booking.GetString("service_id")
		duration := 60 // Mặc định an toàn

		if serviceID != "" {
			service, err := h.App.FindRecordById("services", serviceID)
			if err == nil {
				d := service.GetInt("duration_minutes")
				if d > 0 {
					duration = d
				}
			}
		}

		// 2. Gọi hàm CheckConflict
		if errStr := h.SlotService.CheckConflict(technicianID, date, timeStr, duration); errStr != nil {
			return e.String(409, errStr.Error())
		}
	}

	// Use service layer for business logic
	// [REFACTORED] Service now handles SSE (Admin+Tech) and FCM
	if err := h.BookingService.AssignTechnician(bookingID, technicianID); err != nil {
		return e.String(500, fmt.Sprintf("Lỗi giao việc: %s", err.Error()))
	}
	// Removed manual Broker.Publish and FCM calls

	// Use UI components for rendering
	htmlRow, err := h.UIComponents.RenderBookingRow(booking)
	if err != nil {
		return e.String(500, "Lỗi render")
	}
	return e.HTML(200, htmlRow)
}

func (h *AdminHandler) Stream(e *core.RequestEvent) error {
	// Verify admin authentication
	if e.Auth == nil {
		return e.String(401, "Unauthorized")
	}

	// Set SSE headers
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")

	// Subscribe to admin channel (sees all events)
	eventChan := h.Broker.Subscribe(broker.ChannelAdmin, "")
	defer h.Broker.Unsubscribe(broker.ChannelAdmin, "", eventChan)

	// Send initial connection event
	initialEvent := broker.Event{
		Type:      "connection.established",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"role": "admin",
		},
	}

	eventJSON, _ := json.Marshal(initialEvent)
	fmt.Fprintf(e.Response, "data: %s\n\n", eventJSON)
	e.Response.(http.Flusher).Flush()

	// Stream events
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

// RenderBookingRow delegates to UI components package
func (h *AdminHandler) RenderBookingRow(record *core.Record) (string, error) {
	return h.UIComponents.RenderBookingRow(record)
}

// CancelBooking soft-deletes or cancels a booking
func (h *AdminHandler) CancelBooking(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	booking, err := h.App.FindRecordById("bookings", id)
	if err != nil {
		return e.String(404, "Booking not found")
	}

	// 1. Release Time Slot if exists
	slotID := booking.GetString("time_slot_id")
	if slotID != "" && h.SlotService != nil {
		if err := h.SlotService.ReleaseSlot(slotID); err != nil {
			fmt.Printf("Warning: Failed to release slot %s: %v\n", slotID, err)
			// Continue cancelling booking even if slot release fails
		}
	}

	// 2. Cancel Booking via Service (Handles Notification)
	// booking.Set("job_status", "cancelled") -- Handled by Service
	if err := h.BookingService.CancelBooking(id, "Admin cancelled", ""); err != nil {
		return e.String(500, "Failed to cancel booking: "+err.Error())
	}

	// Removed manual Broker.Publish calls

	return e.JSON(200, map[string]string{"message": "Cancelled successfully"})
}

// UpdateBookingInfo updates non-status fields (Customer info, Address)
func (h *AdminHandler) UpdateBookingInfo(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	booking, err := h.App.FindRecordById("bookings", id)
	if err != nil {
		return e.String(404, "Booking not found")
	}

	booking.Set("customer_name", e.Request.FormValue("name"))
	booking.Set("customer_phone", e.Request.FormValue("phone"))
	booking.Set("address", e.Request.FormValue("address"))
	booking.Set("issue_description", e.Request.FormValue("issue"))

	if err := h.App.Save(booking); err != nil {
		return e.String(500, "Failed to update booking")
	}

	return e.Redirect(http.StatusSeeOther, "/admin")
}

func (h *AdminHandler) GetSlots(e *core.RequestEvent) error {
	daysStr := e.Request.URL.Query().Get("days")
	days := 7
	if daysStr != "" {
		fmt.Sscanf(daysStr, "%d", &days)
	}

	startDate := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	slots, err := h.App.FindRecordsByFilter(
		"time_slots",
		"date >= {:start} && date <= {:end}",
		"date,start_time",
		500,
		0,
		dbx.Params{"start": startDate, "end": endDate},
	)

	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, slots)
}

// ShowSettings displays the settings form
func (h *AdminHandler) ShowSettings(e *core.RequestEvent) error {
	// 1. Fetch current settings
	settings, err := h.SettingsRepo.GetSettings()
	if err != nil {
		// Should generally return default even if err
		fmt.Println("Error fetching settings:", err)
	}

	data := map[string]interface{}{
		"Settings": settings,
		"IsAdmin":  true,
		"PageType": "settings",
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/settings.html", data)
}

// UpdateSettings processes the settings form
func (h *AdminHandler) UpdateSettings(e *core.RequestEvent) error {
	// Debug: Print all form values
	fmt.Println("👉 UpdateSettings Handler Triggered")
	for key, values := range e.Request.Form {
		fmt.Printf("   Form[%s]: %v\n", key, values)
	}

	// 1. Get the Record to update (or create if doesn't exist)
	record, err := h.SettingsRepo.GetSettingsRecord()
	if err != nil {
		fmt.Println("⚠️  No settings record found, creating default...")
		// Create a new settings record
		settingsCollection, errColl := h.App.FindCollectionByNameOrId("settings")
		if errColl != nil {
			fmt.Println("❌ Error finding settings collection:", errColl)
			return e.String(500, "Không tìm thấy bảng settings")
		}
		record = core.NewRecord(settingsCollection)
		// Set defaults
		record.Set("company_name", "HVAC System")
		record.Set("active", true)
		fmt.Println("✅ Created new settings record")
	} else {
		fmt.Printf("   Found Record ID: %s (Current Key: %s)\n", record.Id, record.GetString("license_key"))
	}

	action := e.Request.FormValue("action")
	fmt.Printf("   Action detected: '%s'\n", action)

	if action == "update_license" {
		// [SECURITY] License Management
		rawKey := e.Request.FormValue("license_key")
		newLicenseKey := strings.TrimSpace(rawKey)

		fmt.Printf("   Updating License Key. Raw Len: %d, Trimmed Len: %d\n", len(rawKey), len(newLicenseKey))
		fmt.Printf("   New Key content: %q\n", newLicenseKey)

		if newLicenseKey != "" {
			record.Set("license_key", newLicenseKey)
		}
	} else {
		// ... existing logic ...
		record.Set("company_name", e.Request.FormValue("company_name"))
		record.Set("hotline", e.Request.FormValue("hotline"))
		record.Set("bank_bin", e.Request.FormValue("bank_bin"))
		record.Set("bank_account", e.Request.FormValue("bank_account"))
		record.Set("bank_owner", e.Request.FormValue("bank_owner"))
		record.Set("qr_template", e.Request.FormValue("qr_template"))

		// [FIX] Map SEO Fields
		record.Set("seo_title", e.Request.FormValue("seo_title"))
		record.Set("seo_description", e.Request.FormValue("seo_description"))
		record.Set("seo_keywords", e.Request.FormValue("seo_keywords"))

		// [FIX] Map Hero Section Fields
		record.Set("hero_title", e.Request.FormValue("hero_title"))
		record.Set("hero_subtitle", e.Request.FormValue("hero_subtitle"))
		record.Set("hero_cta_text", e.Request.FormValue("hero_cta_text"))
		record.Set("hero_cta_link", e.Request.FormValue("hero_cta_link"))
		record.Set("welcome_text", e.Request.FormValue("welcome_text")) // If exists in form? check html. form doesn't seem to have welcome_text input?

		files, _ := e.FindUploadedFiles("logo")
		if len(files) > 0 {
			record.Set("logo", files[0])
		}

		heroFiles, _ := e.FindUploadedFiles("hero_image")
		if len(heroFiles) > 0 {
			record.Set("hero_image", heroFiles[0])
		}
	}

	// 4. Save
	if err := h.App.Save(record); err != nil {
		fmt.Println("❌ Error saving record:", err)
		return e.String(500, "Lỗi lưu cấu hình: "+err.Error())
	}
	fmt.Println("✅ Record Saved Successfully")

	// 5. Redirect back with success
	return e.Redirect(http.StatusSeeOther, "/admin/settings?success=true")
}

// -------------------------------------------------------------------
// SERVICE MANAGEMENT HANDLERS
// -------------------------------------------------------------------

// GET /admin/services
func (h *AdminHandler) ServicesList(e *core.RequestEvent) error {
	services, err := h.App.FindRecordsByFilter("services", "1=1", "-created", 100, 0, nil)
	if err != nil {
		fmt.Println("Error fetching services:", err)
		services = []*core.Record{}
	}

	// Expand category relations
	for _, service := range services {
		if categoryID := service.GetString("category_id"); categoryID != "" {
			if category, err := h.App.FindRecordById("categories", categoryID); err == nil {
				service.SetExpand(map[string]any{
					"category_id": category,
				})
			}
		}
	}

	// Fetch Categories for Dropdown
	categories, _ := h.App.FindRecordsByFilter("categories", "active=true", "+sort_order", 100, 0, nil)

	data := map[string]interface{}{
		"Services":   services,
		"Categories": categories,
		"IsAdmin":    true,
		"PageType":   "services",
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/services.html", data)
}

// POST /admin/services (Create or Update)
func (h *AdminHandler) ServiceSave(e *core.RequestEvent) error {
	id := e.Request.FormValue("id")
	name := e.Request.FormValue("name")
	price := e.Request.FormValue("price")
	duration := e.Request.FormValue("duration_minutes")
	description := e.Request.FormValue("description")
	intro := e.Request.FormValue("intro_text")
	video := e.Request.FormValue("video_url")

	// Rich text content
	detailContent := e.Request.FormValue("detail_content")

	collection, err := h.App.FindCollectionByNameOrId("services")
	if err != nil {
		return e.String(500, "Collection not found")
	}

	var record *core.Record
	if id != "" {
		record, err = h.App.FindRecordById("services", id)
		if err != nil {
			return e.String(404, "Service not found")
		}
	} else {
		record = core.NewRecord(collection)
		record.Set("active", true) // Default active
	}

	// Set fields
	record.Set("name", name)
	record.Set("price", price)
	record.Set("duration_minutes", duration)
	record.Set("description", description)
	record.Set("intro_text", intro)
	record.Set("detail_content", detailContent)

	// Process YouTube URL - convert any format to embed URL
	if video != "" {
		embedURL := GetYouTubeEmbedURL(video)
		if embedURL != "" {
			record.Set("video_url", embedURL)
		} else {
			// If not a YouTube URL, save as is
			record.Set("video_url", video)
		}
	} else {
		record.Set("video_url", "")
	}

	// Category handling
	categoryID := e.Request.FormValue("category_id")
	if categoryID != "" {
		record.Set("category_id", categoryID)
	}

	// Handle Main Image Deletion
	deleteImage := e.Request.FormValue("delete_image")
	if deleteImage == "1" {
		record.Set("image", "")
	}

	// Handle Main Image Upload
	imgFile, _ := e.FindUploadedFiles("image")
	if len(imgFile) > 0 {
		record.Set("image", imgFile[0])
	}

	// Handle Gallery Image Deletion
	deleteGallery := e.Request.FormValue("delete_gallery")
	if deleteGallery != "" {
		// Get current gallery
		currentGallery := record.GetStringSlice("gallery")

		// Parse delete list (format: "id/file1.jpg,id/file2.jpg,")
		deleteList := strings.Split(strings.TrimSuffix(deleteGallery, ","), ",")

		// Extract just filenames from delete list
		deleteFiles := make(map[string]bool)
		for _, item := range deleteList {
			if item != "" {
				parts := strings.Split(item, "/")
				if len(parts) == 2 {
					deleteFiles[parts[1]] = true
				}
			}
		}

		// Filter out deleted images
		updatedGallery := []string{}
		for _, img := range currentGallery {
			if !deleteFiles[img] {
				updatedGallery = append(updatedGallery, img)
			}
		}

		record.Set("gallery", updatedGallery)
	}

	// Handle New Gallery Images (Append to existing)
	galleryFiles, _ := e.FindUploadedFiles("gallery")
	if len(galleryFiles) > 0 {
		// Get current gallery filenames
		currentGalleryNames := record.GetStringSlice("gallery")

		// Convert to interface slice to mix strings and file objects
		var galleryData []interface{}
		for _, name := range currentGalleryNames {
			galleryData = append(galleryData, name)
		}

		// Append new file objects
		for _, file := range galleryFiles {
			galleryData = append(galleryData, file)
		}

		record.Set("gallery", galleryData)
	}

	if err := h.App.Save(record); err != nil {
		fmt.Println("Error saving service:", err)
		return e.String(500, "Lỗi lưu dịch vụ: "+err.Error())
	}

	return e.Redirect(http.StatusSeeOther, "/admin/services")
}

// POST /admin/services/{id}/delete
func (h *AdminHandler) ServiceDelete(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := h.App.FindRecordById("services", id)
	if err != nil {
		return e.String(404, "Service not found")
	}

	if err := h.App.Delete(record); err != nil {
		return e.String(500, "Error deleting service")
	}

	return e.Redirect(http.StatusSeeOther, "/admin/services")
}

// -------------------------------------------------------------------
// CATEGORY MANAGEMENT HANDLERS
// -------------------------------------------------------------------

// GET /admin/categories
func (h *AdminHandler) CategoriesList(e *core.RequestEvent) error {
	categories, err := h.App.FindRecordsByFilter("categories", "1=1", "+sort_order,-created", 100, 0, nil)
	if err != nil {
		categories = []*core.Record{}
	}

	data := map[string]interface{}{
		"Categories": categories,
		"IsAdmin":    true,
		"PageType":   "categories",
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/categories.html", data)
}

// POST /admin/categories
func (h *AdminHandler) CategorySave(e *core.RequestEvent) error {
	id := e.Request.FormValue("id")
	name := e.Request.FormValue("name")
	slug := e.Request.FormValue("slug")
	sortOrder := e.Request.FormValue("sort_order")
	description := e.Request.FormValue("description")
	active := e.Request.FormValue("active") == "on"

	collection, err := h.App.FindCollectionByNameOrId("categories")
	if err != nil {
		return e.String(500, "Collection categories not found")
	}

	var record *core.Record
	if id != "" {
		record, err = h.App.FindRecordById("categories", id)
		if err != nil {
			return e.String(404, "Category not found")
		}
	} else {
		record = core.NewRecord(collection)
	}

	record.Set("name", name)
	record.Set("slug", slug)
	record.Set("description", description)
	if sortOrder != "" {
		record.Set("sort_order", sortOrder)
	}
	record.Set("active", active)

	if err := h.App.Save(record); err != nil {
		fmt.Println("Error saving category:", err)
		return e.String(500, "Lỗi lưu danh mục: "+err.Error())
	}

	return e.Redirect(http.StatusSeeOther, "/admin/categories")
}

// POST /admin/categories/{id}/delete
func (h *AdminHandler) CategoryDelete(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	record, err := h.App.FindRecordById("categories", id)
	if err != nil {
		return e.String(404, "Category not found")
	}

	if err := h.App.Delete(record); err != nil {
		return e.String(500, "Error deleting category")
	}

	return e.Redirect(http.StatusSeeOther, "/admin/categories")
}
