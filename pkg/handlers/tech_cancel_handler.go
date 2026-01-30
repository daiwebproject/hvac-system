package handlers

import (
	"time"

	"hvac-system/pkg/broker"

	"github.com/pocketbase/pocketbase/core"
)

// POST /api/tech/bookings/{id}/cancel
func (h *TechHandler) CancelBooking(e *core.RequestEvent) error {
	jobID := e.Request.PathValue("id")
	reason := e.Request.FormValue("reason")

	job, err := h.App.FindRecordById("bookings", jobID)
	if err != nil {
		return e.String(404, "Job not found")
	}

	// Verify this job belongs to the authenticated tech
	if job.GetString("technician_id") != e.Auth.Id {
		return e.String(403, "Unauthorized: This job is not assigned to you")
	}

	// Don't allow cancelling completed jobs
	currentStatus := job.GetString("job_status")
	if currentStatus == "completed" {
		return e.String(400, "Cannot cancel a completed job")
	}

	// Update job status to cancelled
	job.Set("job_status", "cancelled")
	job.Set("cancel_reason", reason)
	job.Set("cancelled_at", time.Now())
	job.Set("cancelled_by", e.Auth.Id) // Track who cancelled

	if err := h.App.Save(job); err != nil {
		return e.String(500, "Failed to cancel job: "+err.Error())
	}

	// Publish cancellation events
	h.Broker.Publish(broker.ChannelCustomer, jobID, broker.Event{
		Type:      "job.cancelled",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"reason":  reason,
			"tech_id": e.Auth.Id,
		},
	})

	h.Broker.Publish(broker.ChannelAdmin, "", broker.Event{
		Type:      "job.cancelled",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"booking_id": jobID,
			"reason":     reason,
			"tech_id":    e.Auth.Id,
		},
	})

	// Return success (client will redirect to job list)
	return e.JSON(200, map[string]interface{}{
		"success": true,
		"message": "Đã hủy công việc",
	})
}
