package handlers

import (
	"fmt"
	"html/template"
	"strconv"
	"time"

	"hvac-system/pkg/broker"
	"hvac-system/pkg/models"
	"hvac-system/pkg/services"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

type WebHandler struct {
	App       *pocketbase.PocketBase
	Templates *template.Template
	Broker    *broker.SegmentedBroker
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
	slotID := e.Request.FormValue("selected_slot")
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

	// Use BookingService
	bookingService := services.NewBookingService(h.App)
	booking, err := bookingService.CreateBooking(services.BookingRequest{
		ServiceID:    serviceID,
		CustomerName: customerName,
		Phone:        customerPhone,
		Address:      address,
		IssueDesc:    issueDesc,
		DeviceType:   deviceType,
		Brand:        brand,
		SlotID:       slotID,      // NEW: Time slot
		BookingTime:  bookingTime, // Legacy fallback
		Lat:          lat,
		Long:         long,
		Files:        files,
	})
	if err != nil {
		return e.String(500, "Lỗi tạo booking: "+err.Error())
	}

	// Publish event to Admin channel
	h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
		Type:      "booking.created",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"booking_id":     booking.Id,
			"customer_name":  booking.GetString("customer_name"),
			"customer_phone": booking.GetString("customer_phone"),
			"service":        booking.GetString("device_type"),
			"booking_time":   booking.GetString("booking_time"),
		},
	})

	// Publish event to Customer channel
	h.Broker.Publish(broker.ChannelCustomer, booking.Id, broker.Event{
		Type:      "booking.confirmed",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"booking_id": booking.Id,
			"status":     "pending",
			"message":    "Booking đã được tạo thành công",
		},
	})

	// If slot was selected, book it
	if slotID != "" {
		slotService := services.NewTimeSlotService(h.App)
		if err := slotService.BookSlot(slotID, booking.Id); err != nil {
			// Slot booking failed, but booking exists - log error or handle
			// For now,continue since booking is created
			fmt.Printf("Error booking slot %s for booking %s: %v\n", slotID, booking.Id, err)
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
