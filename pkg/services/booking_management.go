package services

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// BookingManagementService handles business logic for booking operations
// This separates business logic from HTTP handlers
type BookingManagementService struct {
	app core.App
}

// NewBookingManagementService creates a new booking service
func NewBookingManagementService(app core.App) *BookingManagementService {
	return &BookingManagementService{app: app}
}

// AssignTechnician assigns a technician to a booking
// Business rule: Can only assign if status is "pending" or "assigned"
func (s *BookingManagementService) AssignTechnician(bookingID, technicianID string) error {
	booking, err := s.app.FindRecordById("bookings", bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	// Business rule validation
	currentStatus := booking.GetString("job_status")
	if currentStatus != "pending" && currentStatus != "assigned" {
		return fmt.Errorf("cannot assign technician: job is already %s", currentStatus)
	}

	// Verify technician exists and is active
	technician, err := s.app.FindRecordById("technicians", technicianID)
	if err != nil {
		return fmt.Errorf("technician not found: %w", err)
	}

	if !technician.GetBool("active") {
		return fmt.Errorf("technician is not active")
	}

	// Update booking
	booking.Set("technician_id", technicianID)
	booking.Set("job_status", "assigned")

	if err := s.app.Save(booking); err != nil {
		return fmt.Errorf("failed to save booking: %w", err)
	}

	return nil
}

// RecallToPending resets a job to pending status, releasing resources
func (s *BookingManagementService) RecallToPending(bookingID string, slotService *TimeSlotService) error {
	booking, err := s.app.FindRecordById("bookings", bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	// 1. Release slot if exists
	slotID := booking.GetString("slot_id")
	if slotID != "" {
		if err := slotService.ReleaseSlot(slotID); err != nil {
			// Log warning but continue
			fmt.Printf("Warning: Failed to release slot %s: %v\n", slotID, err)
		}
		booking.Set("slot_id", nil)
	}

	// 2. Clear technician and movement times
	booking.Set("technician_id", nil)
	booking.Set("moving_start_at", nil)
	booking.Set("working_start_at", nil)
	booking.Set("completed_at", nil)

	// 3. pending status
	booking.Set("job_status", "pending")

	if err := s.app.Save(booking); err != nil {
		return fmt.Errorf("failed to save booking: %w", err)
	}

	return nil
}

// UpdateStatus updates the booking status
// Business rule: Validates status transitions
func (s *BookingManagementService) UpdateStatus(bookingID, newStatus string) error {
	booking, err := s.app.FindRecordById("bookings", bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	currentStatus := booking.GetString("job_status")

	// Validate status transition
	if !s.isValidTransition(currentStatus, newStatus) {
		return fmt.Errorf("invalid status transition from %s to %s", currentStatus, newStatus)
	}

	booking.Set("job_status", newStatus)

	if err := s.app.Save(booking); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	return nil
}

// isValidTransition validates if a status transition is allowed
// Business rules for status flow:
// pending -> assigned -> in_transit -> in_progress -> quoting -> completed
func (s *BookingManagementService) isValidTransition(from, to string) bool {
	validTransitions := map[string][]string{
		"pending":     {"assigned", "cancelled"},
		"assigned":    {"in_transit", "cancelled", "pending"}, // Can reassign
		"in_transit":  {"in_progress", "assigned"},            // Can go back if needed
		"in_progress": {"quoting", "completed", "cancelled"},
		"quoting":     {"completed", "in_progress"}, // Can revise quote
		"completed":   {"paid"},
		"cancelled":   {}, // Terminal state
		"paid":        {}, // Terminal state
	}

	allowedStates, exists := validTransitions[from]
	if !exists {
		return false
	}

	for _, allowed := range allowedStates {
		if allowed == to {
			return true
		}
	}

	return false
}

// GetAvailableTechnicians returns technicians not currently on a job
// Business logic: Tech is available if no active jobs assigned
func (s *BookingManagementService) GetAvailableTechnicians() ([]*core.Record, error) {
	// Find all active technicians
	allTechs, err := s.app.FindRecordsByFilter(
		"technicians",
		"active = true",
		"name",
		100,
		0,
		nil,
	)
	if err != nil {
		return nil, err
	}

	// Filter out those with active jobs
	var available []*core.Record
	for _, tech := range allTechs {
		// Check if tech has any non-completed jobs
		// Using dbx.NewExp for complex filter with '!='
		filter := dbx.NewExp("technician_id = {:techId} && job_status != {:completedStatus} && job_status != {:cancelledStatus}", dbx.Params{
			"techId":          tech.Id,
			"completedStatus": "completed",
			"cancelledStatus": "cancelled",
		})
		activeJobs, _ := s.app.CountRecords("bookings", filter)

		if activeJobs == 0 {
			available = append(available, tech)
		}
	}

	return available, nil
}
