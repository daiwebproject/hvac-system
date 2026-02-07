package service

import (
	"context"
	"fmt"
	"hvac-system/internal/core"
	"hvac-system/pkg/broker"
	"log"
	"math"
	"time"
)

type BookingService struct {
	bookingRepo   core.BookingRepository
	techRepo      core.TechnicianRepository
	slotControl   core.TimeSlotControl
	slotRepo      core.TimeSlotRepository // [NEW] To fetch slot details
	notifications core.NotificationService
	settingsRepo  core.SettingsRepository // [NEW]
	broker        *broker.SegmentedBroker
}

func NewBookingService(
	bookingRepo core.BookingRepository,
	techRepo core.TechnicianRepository,
	slotControl core.TimeSlotControl,
	slotRepo core.TimeSlotRepository, // [NEW]
	notifications core.NotificationService,
	settingsRepo core.SettingsRepository, // [NEW]
	eventBroker *broker.SegmentedBroker,
) core.BookingService {
	return &BookingService{
		bookingRepo:   bookingRepo,
		techRepo:      techRepo,
		slotControl:   slotControl,
		slotRepo:      slotRepo, // [NEW]
		notifications: notifications,
		settingsRepo:  settingsRepo,
		broker:        eventBroker,
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

	// [CENTRALIZED NOTIFICATION]
	// 1. SSE to Admin
	if s.broker != nil {
		s.broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "booking.created",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"id":              booking.ID,
				"booking_id":      booking.ID, // Legacy support
				"customer":        booking.CustomerName,
				"customer_name":   booking.CustomerName,
				"phone":           booking.CustomerPhone, // [FIX] Frontend expects 'phone'
				"customer_phone":  booking.CustomerPhone,
				"service":         booking.DeviceType,
				"brand":           booking.Brand,                         // [NEW]
				"device_type":     booking.DeviceType,                    // [NEW] Explicit
				"time":            formatTimeForSSE(booking, s.slotRepo), // Use helper
				"created":         time.Now().Format("15:04"),
				"status":          booking.JobStatus,
				"status_label":    "Chờ xử lý",
				"address":         booking.AddressDetails, // Prioritize details
				"address_details": booking.Address,        // Include both
				"lat":             booking.Lat,
				"long":            booking.Long,
				"issue":           booking.IssueDescription,
			},
		})
	}

	// 2. FCM to Admin (Multicast)
	if s.notifications != nil {
		// Run asynchronously to not block
		go func() {
			// Fetch admin tokens
			settings, err := s.settingsRepo.GetSettings()
			if err == nil {
				if len(settings.AdminFCMTokens) > 0 {
					failedTokens, err := s.notifications.NotifyAdmins(context.Background(), settings.AdminFCMTokens, booking.ID, booking.CustomerName)
					if err != nil {
						log.Printf("❌ [BOOKING_SERVICE] Failed to notify admins: %v", err)
					}
					// [NEW] Cleanup invalid tokens
					if len(failedTokens) > 0 {
						log.Printf("🧹 [BOOKING_SERVICE] Removing %d stale admin tokens...", len(failedTokens))
						for _, t := range failedTokens {
							_ = s.settingsRepo.RemoveAdminToken(t)
						}
					}
				} else {
					// Fallback to Topic
					err = s.notifications.NotifyNewBooking(context.Background(), booking.ID, booking.CustomerName)
				}
			}

			if err != nil {
				// Log error optionally
			}
		}()
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

	// [CENTRALIZED NOTIFICATION]
	// 1. SSE Realtime Event (Tech & Admin)
	if s.broker != nil {
		// Notify Tech
		s.broker.Publish(broker.ChannelTech, technicianID, broker.Event{
			Type:      "job.assigned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"job_id":         booking.ID,
				"booking_id":     booking.ID, // Compatible with old payload
				"customer_name":  booking.CustomerName,
				"customer_phone": booking.CustomerPhone,
				"address":        booking.AddressDetails,
				"booking_time":   booking.BookingTime,
				"device_type":    booking.DeviceType,
			},
		})
		log.Printf("📡 [BOOKING_SERVICE] Published SSE job.assigned to tech %s", technicianID)

		// Notify Admin (Update Kanban/List) - Include full job data for immediate UI update
		s.broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "job.assigned",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"booking_id":   bookingID,
				"id":           bookingID, // Alias for compatibility
				"tech_id":      technicianID,
				"staff_id":     technicianID, // Alias for Kanban
				"tech_name":    tech.Name,
				"customer":     booking.CustomerName,
				"phone":        booking.CustomerPhone,
				"address":      booking.AddressDetails,
				"service":      booking.DeviceType,
				"time":         formatTimeForSSE(booking, s.slotRepo), // [FIX] Include formatted time
				"status":       "assigned",
				"status_label": "Đã giao thợ",
				"lat":          booking.Lat,
				"long":         booking.Long,
			},
		})
	}

	// 2. FCM Push Notification (Tech)
	if s.notifications != nil && tech.FCMToken != "" {
		log.Printf("👉 [BOOKING_SERVICE] Sending FCM to tech %s (TokenLen: %d)", tech.ID, len(tech.FCMToken))
		go func() {
			err := s.notifications.NotifyNewJobAssignment(
				context.Background(),
				tech.FCMToken,
				booking.ID,
				booking.CustomerName,
			)
			if err != nil {
				log.Printf("❌ [BOOKING_SERVICE] FCM Failed: %v", err)
			} else {
				log.Printf("✅ [BOOKING_SERVICE] FCM Sent Successfully to %s", tech.ID)
			}
		}()
	} else {
		log.Printf("⚠️ [BOOKING_SERVICE] Skipped FCM. Service: %v, Token: %s", s.notifications, tech.FCMToken)
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

// Haversine calculates distance in km between two coordinate points
func Haversine(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371 // Earth radius in km
	dLat := (lat2 - lat1) * (math.Pi / 180)
	dLon := (lon2 - lon1) * (math.Pi / 180)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*(math.Pi/180))*math.Cos(lat2*(math.Pi/180))*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c
}

// TechCheckIn verifies technician location and updates status
func (s *BookingService) TechCheckIn(bookingID string, techLat, techLong float64) error {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	// Check distance (allow 0.5km error)
	// If booking has no coordinates (e.g. legacy), we might skip or error.
	// For now assuming booking.Lat/Long are valid if set.
	if booking.Lat != 0 && booking.Long != 0 {
		dist := Haversine(techLat, techLong, booking.Lat, booking.Long)
		if dist > 0.5 {
			return fmt.Errorf("bạn đang cách khách hàng %.2f km (yêu cầu < 0.5 km). Vui lòng đến gần hơn để check-in.", dist)
		}
	}

	// Update DB
	booking.ArrivedAt = time.Now().Format("2006-01-02 15:04:05.000Z") // ISO/PB format
	booking.JobStatus = "arrived"

	if err := s.bookingRepo.Update(booking); err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}

	return nil
}

// CancelBooking updates status to cancelled with a reason
func (s *BookingService) CancelBooking(bookingID, reason, note string) error {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	booking.JobStatus = "cancelled"
	booking.CancelReason = reason
	if note != "" {
		booking.CancelReason += " - Note: " + note
	}

	// Release slot if applicable
	if booking.SlotID != nil && *booking.SlotID != "" {
		if s.slotControl != nil {
			_ = s.slotControl.ReleaseSlot(*booking.SlotID)
		}
		booking.SlotID = nil
	}

	if err := s.bookingRepo.Update(booking); err != nil {
		return fmt.Errorf("failed to cancel booking: %w", err)
	}

	// [CENTRALIZED NOTIFICATION]
	if s.broker != nil {
		// Notify Admin
		s.broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "booking.cancelled",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"id":     bookingID,
				"reason": reason,
				"note":   note,
			},
		})

		// Notify Tech
		if booking.TechnicianID != "" {
			s.broker.Publish(broker.ChannelTech, booking.TechnicianID, broker.Event{
				Type:      "job.cancelled",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"booking_id": bookingID,
					"reason":     reason,
				},
			})
		}
	}

	// [NEW] FCM to Admin (Multicast)
	if s.notifications != nil {
		go func() {
			// Fetch admin tokens
			settings, err := s.settingsRepo.GetSettings()
			if err == nil && len(settings.AdminFCMTokens) > 0 {
				failedTokens, err := s.notifications.NotifyAdminsBookingCancelled(context.Background(), settings.AdminFCMTokens, bookingID, booking.CustomerName, reason, note)
				if err != nil {
					log.Printf("❌ [BOOKING_SERVICE] Failed to notify admins of cancellation: %v", err)
				}
				// [NEW] Cleanup invalid tokens
				if len(failedTokens) > 0 {
					log.Printf("🧹 [BOOKING_SERVICE] Removing %d stale admin tokens during cancel...", len(failedTokens))
					for _, t := range failedTokens {
						_ = s.settingsRepo.RemoveAdminToken(t)
					}
				}
			}
		}()
	}

	return nil
}

// RescheduleBooking updates the booking time
func (s *BookingService) RescheduleBooking(bookingID, newTime string) error {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return fmt.Errorf("booking not found: %w", err)
	}

	booking.BookingTime = newTime
	// Reset status if it was completed/cancelled? Assuming active job.
	// We might want to keep it assigned.

	if err := s.bookingRepo.Update(booking); err != nil {
		return fmt.Errorf("failed to reschedule booking: %w", err)
	}
	return nil
}
