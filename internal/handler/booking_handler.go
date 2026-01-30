package handler

import (
	"errors"
	domain "hvac-system/internal/core"

	pbCore "github.com/pocketbase/pocketbase/core"
)

type BookingHandler struct {
	service domain.BookingService
}

func NewBookingHandler(service domain.BookingService) *BookingHandler {
	return &BookingHandler{service: service}
}

// AssignJob handles POST /bookings/{id}/assign
func (h *BookingHandler) AssignJob(e *pbCore.RequestEvent) error {
	bookingID := e.Request.PathValue("id")
	techID := e.Request.FormValue("technician_id")

	if bookingID == "" || techID == "" {
		return e.JSON(400, map[string]string{"error": "Missing booking ID or technician ID"})
	}

	if err := h.service.AssignTechnician(bookingID, techID); err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]string{"message": "Assigned successfully"})
}

// UpdateStatus handles POST /api/bookings/{id}/status
func (h *BookingHandler) UpdateStatus(e *pbCore.RequestEvent) error {
	id := e.Request.PathValue("id")
	status := e.Request.FormValue("status")

	if id == "" || status == "" {
		return e.JSON(400, map[string]string{"error": "Missing information"})
	}

	if status == "pending" {
		if err := h.service.RecallToPending(id); err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(200, map[string]string{"message": "Recalled to pending"})
	}

	if err := h.service.UpdateStatus(id, status); err != nil {
		if errors.Is(err, errors.New("invalid status")) { // Simplified error check
			return e.JSON(400, map[string]string{"error": err.Error()})
		}
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]string{"message": "Status updated"})
}
