package service

import (
	"fmt"
	"hvac-system/internal/core"
	"time"
)

type TimeSlotService struct {
	slotRepo    core.TimeSlotRepository
	bookingRepo core.BookingRepository
	serviceRepo core.ServiceRepository
}

func NewTimeSlotService(
	slotRepo core.TimeSlotRepository,
	bookingRepo core.BookingRepository,
	serviceRepo core.ServiceRepository,
) core.TimeSlotControl {
	return &TimeSlotService{
		slotRepo:    slotRepo,
		bookingRepo: bookingRepo,
		serviceRepo: serviceRepo,
	}
}

func (s *TimeSlotService) ReleaseSlot(slotID string) error {
	slot, err := s.slotRepo.GetByID(slotID)
	if err != nil {
		return err
	}

	if slot.CurrentBookings > 0 {
		slot.CurrentBookings--
		slot.IsBooked = false // Re-open
		return s.slotRepo.Update(slot)
	}
	return nil
}

func (s *TimeSlotService) BookSlot(slotID, bookingID string) error {
	slot, err := s.slotRepo.GetByID(slotID)
	if err != nil {
		return err
	}

	if slot.CurrentBookings >= slot.MaxCapacity {
		return fmt.Errorf("slot is fully booked")
	}

	slot.CurrentBookings++
	if slot.CurrentBookings >= slot.MaxCapacity {
		slot.IsBooked = true
	}

	return s.slotRepo.Update(slot)
}

func (s *TimeSlotService) CheckConflict(techID, date, timeStr string, durationMin int, newSlotID string) error {
	const TravelBufferMinutes = 30

	// 1. Calculate New Job Times
	newStart, err := time.Parse("2006-01-02 15:04", date+" "+timeStr)
	if err != nil {
		return fmt.Errorf("invalid date/time: %v", err)
	}
	newEnd := newStart.Add(time.Duration(durationMin) * time.Minute)

	// 2. Fetch existing jobs for technician
	bookings, err := s.bookingRepo.FindScheduledByTechnician(techID)
	if err != nil {
		return fmt.Errorf("failed to fetch existing jobs: %w", err)
	}

	// 3. Iterate and check overlaps
	for _, job := range bookings {
		// [NEW] Check Slot overlap if both have slot IDs
		if newSlotID != "" && job.SlotID != nil {
			if *job.SlotID == newSlotID {
				// Only if dates match (Job SlotID might be reused across days? -> Slot ID is usually unique record ID in PB)
				// Assuming Slot ID is unique record ID from time_slots collection, checks are safe.
				// But we should double check date just in case.
				// With current architecture, Slot ID is unique globally.
				return fmt.Errorf("Slot conflict: Technician already assigned to this slot")
			}
		}

		// Parse job time (using booking_time field which format is expected to be "YYYY-MM-DD HH:MM")
		if len(job.BookingTime) < 16 {
			continue
		}

		jobStart, err := time.Parse("2006-01-02 15:04", job.BookingTime)
		if err != nil {
			continue
		}

		// Only check jobs on same day
		if jobStart.Format("2006-01-02") != date {
			continue
		}

		// Determine duration
		existingDuration := 60
		if job.ServiceID != "" {
			svc, err := s.serviceRepo.GetByID(job.ServiceID)
			if err == nil && svc.DurationMinutes > 0 {
				existingDuration = svc.DurationMinutes
			}
		}

		jobEnd := jobStart.Add(time.Duration(existingDuration) * time.Minute)

		// Check overlap with buffers
		bufferedStart := jobStart.Add(-time.Duration(TravelBufferMinutes) * time.Minute)
		bufferedEnd := jobEnd.Add(time.Duration(TravelBufferMinutes) * time.Minute)

		if newStart.Before(bufferedEnd) && newEnd.After(bufferedStart) {
			return fmt.Errorf(
				"Conflict: Technician busy from %s to %s (Service: %dm + 30m buffer)",
				jobStart.Format("15:04"),
				jobEnd.Format("15:04"),
				existingDuration,
			)
		}
	}

	return nil
}
