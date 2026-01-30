package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

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
	BookingService   *services.BookingManagementService
	SlotService      *services.TimeSlotService
	AnalyticsService *services.AnalyticsService
	UIComponents     *ui.Components
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


// Dashboard renders the admin dashboard with Kanban board
func (h *AdminHandler) Dashboard(e *core.RequestEvent) error {
	// 1. Fetch active bookings
	bookings, err := h.App.FindRecordsByFilter(
		"bookings",
		"job_status != 'cancelled'",
		"-id",
		100,
		0,
		nil,
	)
	if err != nil {
		return e.String(500, "Lỗi load booking: "+err.Error())
	}

	// Fetch technicians
	technicians, _ := h.App.FindRecordsByFilter("technicians", "active=true", "name", 100, 0, nil)

	// 2. Fetch Stats & Analytics (Giữ nguyên logic cũ)
	var revenue float64
	invoices, _ := h.App.FindRecordsByFilter("invoices", "status='paid'", "", 0, 0, nil)
	for _, inv := range invoices {
		revenue += inv.GetFloat("total_amount")
	}

	today := time.Now().Format("2006-01-02")
	bookingsToday, _ := h.App.CountRecords("bookings", dbx.NewExp("created >= {:date}", dbx.Params{"date": today}))
	activeTechs, _ := h.App.CountRecords("technicians", dbx.NewExp("verified=true"))
	pendingCount, _ := h.App.CountRecords("bookings", dbx.NewExp("job_status = 'pending'"))
	completedCount, _ := h.App.CountRecords("bookings", dbx.NewExp("job_status = 'completed'"))

	completionRate := 0.0
	if bookingsToday > 0 {
		completionRate = (float64(completedCount) / float64(bookingsToday)) * 100
	}

	revenueStats, _ := h.AnalyticsService.GetRevenueLast7Days()
	topTechs, _ := h.AnalyticsService.GetTopTechnicians(5)

	// 3. Serialize bookings to JSON (Cải tiến hiển thị thời gian)
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
	}

	var bookingsJSON []BookingJSON
	for _, b := range bookings {
		// Xử lý tên dịch vụ
		serviceName := b.GetString("device_type")
		if serviceName == "" {
			serviceName = "Kiểm tra / Khác"
		}

		// --- [LOGIC MỚI] Xử lý hiển thị thời gian (Date + Time Range) ---
		rawTime := b.GetString("booking_time")
		displayTime := rawTime // Mặc định nếu parse lỗi

		// PocketBase lưu dạng UTC: "2006-01-02 15:04:05.000Z"
		// Parse thử 2 định dạng phổ biến
		parsedTime, err := time.Parse("2006-01-02 15:04:05.000Z", rawTime)
		if err != nil {
			parsedTime, err = time.Parse("2006-01-02 15:04:05", rawTime)
		}

		if err == nil {
			// Giả sử mỗi ca làm 2 tiếng (Bạn có thể sửa số 2 thành số khác tùy ý)
			endTime := parsedTime.Add(2 * time.Hour)

			// Format: Ngày/Tháng Giờ:Phút - Giờ:Phút (VD: 30/01 10:00 - 12:00)
			displayTime = fmt.Sprintf("%02d/%02d %02d:%02d - %02d:%02d",
				parsedTime.Day(),
				parsedTime.Month(),
				parsedTime.Hour(), parsedTime.Minute(),
				endTime.Hour(), endTime.Minute(),
			)
		}
		// -------------------------------------------------------------

		bookingsJSON = append(bookingsJSON, BookingJSON{
			ID:             b.Id,
			Customer:       b.GetString("customer_name"),
			StaffID:        b.GetString("technician_id"),
			Service:        serviceName,
			Time:           displayTime,            // Đã format đẹp
			Created:        b.GetString("created"), // Lấy thời gian tạo
			Status:         b.GetString("job_status"),
			StatusLabel:    b.GetString("job_status"),
			Phone:          b.GetString("customer_phone"),
			AddressDetails: b.GetString("address_details"),
			Address:        b.GetString("address"),
			Lat:            b.GetFloat("lat"),
			Long:           b.GetFloat("long"),
			Issue:          b.GetString("issue_description"),
		})
	}

	bookingsJSONBytes, _ := json.Marshal(bookingsJSON)

	data := map[string]interface{}{
		"Bookings":       bookings,
		"BookingsJSON":   template.JS(string(bookingsJSONBytes)),
		"Technicians":    technicians,
		"TotalRevenue":   revenue,
		"BookingsToday":  bookingsToday,
		"ActiveTechs":    activeTechs,
		"Pending":        pendingCount,
		"Completed":      completedCount,
		"CompletionRate": completionRate,
		"RevenueStats":   revenueStats,
		"TopTechs":       topTechs,
		"IsAdmin":        true,
		"PageType":       "admin_dashboard",
	}

	return RenderPage(h.Templates, e, "layouts/admin.html", "admin/dashboard.html", data)
}

func (h *AdminHandler) UpdateBookingStatus(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	status := e.Request.FormValue("status")

	if id == "" || status == "" {
		return e.String(400, "Missing ID or Status")
	}

	// --- [LOGIC QUAN TRỌNG] Xử lý khi kéo về Pending (Thu hồi việc) ---
	if status == "pending" {
		booking, err := h.App.FindRecordById("bookings", id)
		if err != nil {
			return e.String(404, "Booking not found")
		}

		// 1. Trả slot (nếu có)
		slotID := booking.GetString("slot_id")
		if slotID != "" {
			if err := h.SlotService.ReleaseSlot(slotID); err != nil {
				fmt.Printf("Warning: Failed to release slot %s: %v\n", slotID, err)
			}
			// [BẮT BUỘC] Dùng nil (không dùng "")
			booking.Set("slot_id", nil)
		}

		// 2. Xóa thợ và ngày giờ làm việc
		// [BẮT BUỘC] Dùng nil cho tất cả các trường này
		booking.Set("technician_id", nil)
		booking.Set("moving_start_at", nil)
		booking.Set("working_start_at", nil)
		booking.Set("completed_at", nil)

		// 3. Cập nhật trạng thái
		booking.Set("job_status", "pending")

		// 4. Lưu vào DB
		if err := h.App.Save(booking); err != nil {
			fmt.Println("Lỗi lưu DB (Pending):", err) // Log lỗi ra để xem
			return e.String(500, "Lỗi cập nhật DB: "+err.Error())
		}

		// [BẮT BUỘC] RETURN NGAY TẠI ĐÂY
		// Tuyệt đối không để code chạy xuống dòng BookingService bên dưới
		return e.JSON(200, map[string]string{"message": "Đã thu hồi về Pending"})
	}

	// --- Logic cho các trạng thái khác (Assigned, Working, Completed) ---
	// Chỉ chạy dòng này khi status KHÔNG PHẢI là pending
	if err := h.BookingService.UpdateStatus(id, status); err != nil {
		return e.String(500, err.Error())
	}

	return e.JSON(200, map[string]string{"message": "Success"})
}
func (h *AdminHandler) AssignJob(e *core.RequestEvent) error {
	bookingID := e.Request.PathValue("id")

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

		// 2. Gọi hàm CheckConflict đã nâng cấp
		// Truyền duration thực tế vào thay vì số cứng
		if errStr := h.SlotService.CheckConflict(technicianID, date, timeStr, duration); errStr != nil {
			// Trả về lỗi 409 Conflict để UI hiển thị thông báo đỏ
			return e.String(409, errStr.Error())
		}
	}

	// Use service layer for business logic
	if err := h.BookingService.AssignTechnician(bookingID, technicianID); err != nil {
		return e.String(500, fmt.Sprintf("Lỗi giao việc: %s", err.Error()))
	}

	// Fetch updated booking for event data
	// Note: We ignore error here as we just verified it exists or it was just updated
	if updatedBooking, err := h.App.FindRecordById("bookings", bookingID); err == nil {
		booking = updatedBooking

		// Publish to Admin channel
		h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "job.assigned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"booking_id": bookingID,
				"tech_id":    technicianID,
				"customer":   booking.GetString("customer_name"),
			},
		})

		// Publish to Tech channel
		h.Broker.Publish(broker.ChannelTech, technicianID, broker.Event{
			Type:      "job.assigned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"booking_id":     bookingID,
				"customer_name":  booking.GetString("customer_name"),
				"customer_phone": booking.GetString("customer_phone"),
				"address":        booking.GetString("address"),
				"booking_time":   booking.GetString("booking_time"),
				"device_type":    booking.GetString("device_type"),
			},
		})
	}

	// Fetch updated booking for rendering
	// booking already fetched above

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
// Kept for backward compatibility, but should migrate callers to use ui.Components directly
func (h *AdminHandler) RenderBookingRow(record *core.Record) (string, error) {
	return h.UIComponents.RenderBookingRow(record)
}

// CancelBooking soft-deletes or cancels a booking
// POST /admin/bookings/{id}/cancel
func (h *AdminHandler) CancelBooking(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	booking, err := h.App.FindRecordById("bookings", id)
	if err != nil {
		return e.String(404, "Booking not found")
	}

	booking.Set("job_status", "cancelled")
	if err := h.App.Save(booking); err != nil {
		return e.String(500, "Failed to cancel booking")
	}

	// Notify system
	h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
		Type:      "booking.cancelled",
		Timestamp: time.Now().Unix(),
		Data:      map[string]interface{}{"id": id},
	})

	return e.JSON(200, map[string]string{"message": "Cancelled successfully"})
}

// UpdateBookingInfo updates non-status fields (Customer info, Address)
// POST /admin/bookings/{id}/update
func (h *AdminHandler) UpdateBookingInfo(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	booking, err := h.App.FindRecordById("bookings", id)
	if err != nil {
		return e.String(404, "Booking not found")
	}

	// Update fields
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

	// Tính toán ngày bắt đầu (hôm nay) và kết thúc
	startDate := time.Now().Format("2006-01-02")
	endDate := time.Now().AddDate(0, 0, days).Format("2006-01-02")

	// Query database lấy slots
	// Sắp xếp theo ngày và giờ bắt đầu
	slots, err := h.App.FindRecordsByFilter(
		"time_slots",
		"date >= {:start} && date <= {:end}",
		"date,start_time",
		500, // Tăng limit an toàn
		0,
		dbx.Params{"start": startDate, "end": endDate},
	)

	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, slots)
}
