package service

import (
	"fmt"
	"hvac-system/internal/core"
)

type BookingService struct {
	bookingRepo core.BookingRepository
	techRepo    core.TechnicianRepository
	slotControl core.TimeSlotControl
}

func NewBookingService(
	bookingRepo core.BookingRepository,
	techRepo core.TechnicianRepository,
	slotControl core.TimeSlotControl,
) core.BookingService {
	return &BookingService{
		bookingRepo: bookingRepo,
		techRepo:    techRepo,
		slotControl: slotControl,
	}
}

func (s *BookingService) CreateBooking(req *core.BookingRequest) (*core.Booking, error) {
	booking := &core.Booking{
		ServiceID:        req.ServiceID,
		CustomerName:     req.CustomerName,
		CustomerPhone:    req.Phone,
		Address:          req.AddressDetails, // Logic mapping
		AddressDetails:   req.Address,
		IssueDescription: req.IssueDesc,
		DeviceType:       req.DeviceType,
		Brand:            req.Brand,
		JobStatus:        "pending",
		Lat:              req.Lat,
		Long:             req.Long,
	}

	if req.SlotID != "" {
		slotID := req.SlotID
		booking.SlotID = &slotID
	} else if req.BookingTime != "" {
		booking.BookingTime = req.BookingTime
	}

	if err := s.bookingRepo.Create(booking, req.Files); err != nil {
		return nil, err
	}

	return booking, nil
}

func (s *BookingService) AssignTechnician(bookingID, technicianID string) error {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	// Validation
	if booking.JobStatus != "pending" && booking.JobStatus != "assigned" {
		return fmt.Errorf("cannot assign technician: job is already %s", booking.JobStatus)
	}

	tech, err := s.techRepo.GetByID(technicianID)
	if err != nil {
		return fmt.Errorf("technician not found: %w", err)
	}
	if !tech.Active {
		return fmt.Errorf("technician is not active")
	}

	// Update Domain Model
	booking.TechnicianID = technicianID
	booking.JobStatus = "assigned"

	// Persist
	if err := s.bookingRepo.Update(booking); err != nil {
		return fmt.Errorf("failed to save assignment: %w", err)
	}

	return nil
}

func (s *BookingService) RecallToPending(bookingID string) error {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	// Release slot
	if booking.SlotID != nil && *booking.SlotID != "" {
		if s.slotControl != nil {
			// Ignore error for now, just log if posssible
			_ = s.slotControl.ReleaseSlot(*booking.SlotID)
		}
		booking.SlotID = nil
	}

	// Clear Assignments
	booking.TechnicianID = ""
	booking.JobStatus = "pending"

	if err := s.bookingRepo.Update(booking); err != nil {
		return fmt.Errorf("failed to recall booking: %w", err)
	}

	return nil
}

func (s *BookingService) UpdateStatus(bookingID, status string) error {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	// Validation logic for transitions could be here

	booking.JobStatus = status
	if err := s.bookingRepo.Update(booking); err != nil {
		return err
	}

	return nil
}
