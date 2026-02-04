package handlers

import (
	"fmt"
	"math"
	"strconv"

	"github.com/pocketbase/pocketbase/core"
)

// haversine calculates distance in km between two coordinate points
func haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// CancelBooking handles job cancellation or rescheduling
// POST /api/tech/bookings/{id}/cancel
func (h *TechHandler) CancelBooking(e *core.RequestEvent) error {
	bookingID := e.Request.PathValue("id")
	reason := e.Request.FormValue("reason")
	note := e.Request.FormValue("note")
	newTime := e.Request.FormValue("new_time")

	if reason == "" {
		return e.JSON(400, map[string]string{"error": "Vui lòng chọn lý do"})
	}

	// 1. Handle Reschedule
	if reason == "reschedule" {
		if newTime == "" {
			return e.JSON(400, map[string]string{"error": "Vui lòng chọn thời gian mới"})
		}
		// Call Service Reschedule
		err := h.BookingService.RescheduleBooking(bookingID, newTime)
		if err != nil {
			return e.JSON(500, map[string]string{"error": err.Error()})
		}
		return e.JSON(200, map[string]string{"status": "success", "message": "Đã hẹn lại lịch với khách"})
	}

	// 2. Handle Cancellation (Customer not home, etc)
	if reason == "customer_not_home" {
		files, _ := e.FindUploadedFiles("evidence")
		if len(files) == 0 {
			return e.JSON(400, map[string]string{"error": "Cần chụp ảnh cửa nhà khách làm bằng chứng"})
		}

		// Create a report for this evidence
		jobReports, _ := h.App.FindCollectionByNameOrId("job_reports")
		report := core.NewRecord(jobReports)
		report.Set("booking_id", bookingID)
		report.Set("tech_id", e.Auth.Id)
		report.Set("photo_notes", "Bằng chứng khách vắng nhà")

		fileSlice := make([]any, len(files))
		for i, f := range files {
			fileSlice[i] = f
		}
		report.Set("after_images", fileSlice) // Using after_images as generic 'evidence'
		h.App.Save(report)                    // Ignore error for now or log it
	}

	// 3. Calculate Distance for Verification (if provided)
	latStr := e.Request.FormValue("lat")
	longStr := e.Request.FormValue("long")
	if latStr != "" && longStr != "" {
		booking, err := h.App.FindRecordById("bookings", bookingID)
		if err == nil {
			techLat, _ := strconv.ParseFloat(latStr, 64)
			techLong, _ := strconv.ParseFloat(longStr, 64)
			custLat := booking.GetFloat("lat")
			custLong := booking.GetFloat("long")

			if custLat != 0 && custLong != 0 {
				dist := haversine(techLat, techLong, custLat, custLong)
				note = fmt.Sprintf("%s [Vị trí thợ cách khách: %.2f km]", note, dist)
			}
		}
	}

	// Call Service Cancel
	err := h.BookingService.CancelBooking(bookingID, reason, note)
	if err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	return e.JSON(200, map[string]string{"status": "success", "message": "Đã hủy công việc"})
}
