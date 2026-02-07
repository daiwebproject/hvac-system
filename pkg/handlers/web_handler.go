package handlers

import (
	"fmt"
	"html/template"
	"strconv"

	"hvac-system/internal/adapter/repository"
	domain "hvac-system/internal/core"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/models"
	"hvac-system/pkg/notification"
	"hvac-system/pkg/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type WebHandler struct {
	App            *pocketbase.PocketBase
	Templates      *template.Template
	Broker         *broker.SegmentedBroker
	SettingsRepo   *repository.SettingsRepo // [NEW]
	FCMService     *notification.FCMService // [NEW]
	BookingService domain.BookingService    // [NEW] Internal Service
}

// 1. Trang chủ - Landing Page
func (h *WebHandler) Index(e *core.RequestEvent) error {
	// Fetch services
	var services []models.Service
	err := h.App.DB().
		NewQuery("SELECT * FROM services WHERE active=true ORDER BY id DESC").
		All(&services)
	if err != nil {
		fmt.Println("Error fetching services:", err)
		services = []models.Service{}
	}

	// Render template
	// Use atomic design helper
	return RenderPage(h.Templates, e, "layouts/base.html", "public/index.html", map[string]interface{}{
		"Services": services,
	})
}

// 2. API Đặt lịch
func (h *WebHandler) BookService(e *core.RequestEvent) error {
	// Parse form values
	serviceID := e.Request.FormValue("service_id")
	customerName := e.Request.FormValue("customer_name")
	customerPhone := e.Request.FormValue("customer_phone")
	address := e.Request.FormValue("address")
	issueDesc := e.Request.FormValue("issue_description")
	deviceType := e.Request.FormValue("device_type")
	brand := e.Request.FormValue("brand")

	// Time slot ID (new) or legacy booking_time
	slotID := e.Request.FormValue("slot_id")           // [FIX] Match frontend key
	bookingTime := e.Request.FormValue("booking_time") // Fallback for old flow

	// Location
	latStr := e.Request.FormValue("lat")
	longStr := e.Request.FormValue("long")
	lat, _ := strconv.ParseFloat(latStr, 64)
	long, _ := strconv.ParseFloat(longStr, 64)

	// Validation
	if customerName == "" || customerPhone == "" || address == "" {
		return e.String(400, "Thiếu thông tin bắt buộc")
	}

	// Handle File Uploads
	files, _ := e.FindUploadedFiles("client_images")

	// Use BookingService (Centralized)
	// [REFACTORED] Switched to internal domain service which handles SSE/FCM
	booking, err := h.BookingService.CreateBooking(&domain.BookingRequest{
		ServiceID:      serviceID,
		CustomerName:   customerName,
		Phone:          customerPhone,
		AddressDetails: address, // Map to Details as per Admin implementation
		IssueDesc:      issueDesc,
		DeviceType:     deviceType,
		Brand:          brand,
		SlotID:         slotID,      // NEW: Time slot
		BookingTime:    bookingTime, // Legacy fallback
		Lat:            lat,
		Long:           long,
		Files:          files,
	})
	if err != nil {
		return e.String(500, "Lỗi tạo booking: "+err.Error())
	}

	// [REMOVED] Duplicate SSE and FCM logic - handled by Service

	// If slot was selected, book it
	if slotID != "" {
		slotService := services.NewTimeSlotService(h.App)
		if err := slotService.BookSlot(slotID, booking.ID); err != nil {
			// Slot booking failed, but booking exists - log error or handle
			// For now,continue since booking is created
			fmt.Printf("Error booking slot %s for booking %s: %v\n", slotID, booking.ID, err)
		}
	}

	return e.HTML(200, `
        <div class="alert alert-success shadow-lg">
            <div>
				<i class="fa-solid fa-check-circle"></i>
				<span>Đã nhận yêu cầu! Kỹ thuật viên sẽ gọi lại trong 5 phút.</span>
			</div>
        </div>
    `)
}

// Booking Page Handler
func (h *WebHandler) BookingPage(e *core.RequestEvent) error {
	services, _ := h.App.FindRecordsByFilter("services", "active=true", "price", 100, 0)
	// Use layout inheritance
	// Note: We are using "public/booking_wizard.html" which we just created
	return RenderPage(h.Templates, e, "layouts/base.html", "public/booking_wizard.html", map[string]interface{}{
		"Services": services,
	})
}
