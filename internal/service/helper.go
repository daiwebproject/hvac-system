package service

import (
	"fmt"
	"hvac-system/internal/core"
	"time"
)

// Helper to format time for SSE
// Resolves SlotID to actual time range if present
func formatTimeForSSE(booking *core.Booking, slotRepo core.TimeSlotRepository) string {
	// 1. Try Slot ID
	if booking.SlotID != nil && *booking.SlotID != "" && slotRepo != nil {
		slot, err := slotRepo.GetByID(*booking.SlotID)
		if err == nil && slot != nil {
			// Format: "08:00 - 10:00 07/02"
			if tDate, err := time.Parse("2006-01-02", slot.Date); err == nil {
				return fmt.Sprintf("%s - %s %02d/%02d", slot.StartTime, slot.EndTime, tDate.Day(), tDate.Month())
			}
			return fmt.Sprintf("%s - %s", slot.StartTime, slot.EndTime)
		}
	}

	// 2. Fallback to BookingTime
	// Support both DB format "YYYY-MM-DD HH:MM:SS.000Z" and "YYYY-MM-DD HH:MM"
	rawTime := booking.BookingTime
	if rawTime == "" {
		return ""
	}

	parsedTime, err := time.Parse("2006-01-02 15:04:05.000Z", rawTime)
	if err != nil {
		parsedTime, err = time.Parse("2006-01-02 15:04", rawTime)
	}

	if err == nil {
		endTime := parsedTime.Add(2 * time.Hour)
		return fmt.Sprintf("%s - %s %02d/%02d",
			parsedTime.Format("15:04"),
			endTime.Format("15:04"),
			parsedTime.Day(), parsedTime.Month(),
		)
	}

	return rawTime
}
