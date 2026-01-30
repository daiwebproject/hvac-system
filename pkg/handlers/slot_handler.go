package handlers

import (
	"hvac-system/pkg/services"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// SlotHandler handles time slot API requests
type SlotHandler struct {
	App         core.App
	SlotService *services.TimeSlotService
}

// GetAvailableSlots returns available slots for a date
// GET /api/slots/available?date=2026-01-28
func (h *SlotHandler) GetAvailableSlots(e *core.RequestEvent) error {
	dateStr := e.Request.URL.Query().Get("date")
	if dateStr == "" {
		// Default to tomorrow
		dateStr = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}

	slots, err := h.SlotService.GetAvailableSlots(dateStr)
	if err != nil {
		return e.JSON(400, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, slots)
}

// GenerateSlots creates default slots for a date (Admin only)
// POST /api/slots/generate
// Body: {"date": "2026-01-28", "tech_count": 3}
func (h *SlotHandler) GenerateSlots(e *core.RequestEvent) error {
	var req struct {
		Date      string `json:"date"`
		TechCount int    `json:"tech_count"`
	}

	if err := e.BindBody(&req); err != nil {
		return e.JSON(400, map[string]string{"error": "Invalid request"})
	}

	if req.Date == "" || req.TechCount == 0 {
		return e.JSON(400, map[string]string{"error": "date and tech_count required"})
	}

	if err := h.SlotService.GenerateDefaultSlots(req.Date, req.TechCount); err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]string{"message": "Slots generated successfully"})
}
