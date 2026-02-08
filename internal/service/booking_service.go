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

		// [FIX] Populate BookingTime from Slot
		if s.slotRepo != nil {
			slot, err := s.slotRepo.GetByID(slotID)
			if err == nil && slot != nil {
				// Format: YYYY-MM-DD HH:MM
				booking.BookingTime = fmt.Sprintf("%s %s", slot.Date, slot.StartTime)
			} else {
				log.Printf("⚠️ [BOOKING_SERVICE] Failed to fetch slot %s: %v", slotID, err)
				// Fallback to req.BookingTime if available
				if req.BookingTime != "" {
					booking.BookingTime = req.BookingTime
				}
			}
		}
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

				// [FIX] Auto-cleanup invalid tokens
				if err.Error() == "token_invalid" {
					log.Printf("🧹 [BOOKING_SERVICE] Cleaning up invalid token for tech %s", tech.ID)
					tech.FCMToken = ""
					// We need to update the tech record.
					// techRepo interface might need Update method, or we rely on FindRecordById in app.
					// Since we have techRepo, let's use it if available or fallback.
					if err := s.techRepo.Update(tech); err != nil {
						log.Printf("⚠️ [BOOKING_SERVICE] Failed to clear invalid token: %v", err)
					}
				}
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

	// Capture old tech ID for notification
	oldTechID := booking.TechnicianID

	// Clear Assignments
	booking.TechnicianID = ""
	booking.JobStatus = "pending"

	if err := s.bookingRepo.Update(booking); err != nil {
		return fmt.Errorf("failed to recall booking: %w", err)
	}

	// [CENTRALIZED NOTIFICATION]
	if s.broker != nil {
		// Notify Old Tech (SSE) - Trigger list reload (job will disappear)
		if oldTechID != "" {
			s.broker.Publish(broker.ChannelTech, oldTechID, broker.Event{
				Type:      "job.status_changed",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"job_id": bookingID,
					"status": "pending", // Will cause it to be removed from tech view
				},
			})
		}

		// Notify Admin (SSE)
		s.broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "job.status_changed",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"booking_id": bookingID,
				"status":     "pending",
				"tech_id":    oldTechID, // Inform admin it was unassigned
			},
		})
	}

	// Notify Old Tech (FCM)
	if s.notifications != nil && oldTechID != "" {
		go func() {
			tech, err := s.techRepo.GetByID(oldTechID)
			if err == nil && tech.FCMToken != "" {
				// Use "cancelled" or custom message for unassignment?
				// "pending" maps to "Chờ duyệt", might be confusing.
				// "cancelled" maps to "Đơn hàng đã hủy".
				// Let's use "cancelled" for FCM to indicate it's gone from their list.
				err := s.notifications.NotifyJobStatusChange(context.Background(), tech.FCMToken, bookingID, "cancelled")
				if err != nil && err.Error() == "token_invalid" {
					log.Printf("🧹 [BOOKING_SERVICE] Cleaning up invalid token for tech %s in RecallToPending", tech.ID)
					tech.FCMToken = ""
					if err := s.techRepo.Update(tech); err != nil {
						log.Printf("⚠️ [BOOKING_SERVICE] Failed to clear invalid token: %v", err)
					}
				}
			}
		}()
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

	// [CENTRALIZED NOTIFICATION]
	if s.broker != nil {
		// Notify Tech (SSE)
		if booking.TechnicianID != "" {
			s.broker.Publish(broker.ChannelTech, booking.TechnicianID, broker.Event{
				Type:      "job.status_changed",
				Timestamp: time.Now().Unix(),
				Data: map[string]interface{}{
					"job_id": bookingID,
					"status": status,
				},
			})
		}
		// Notify Admin (SSE)
		s.broker.Publish(broker.ChannelAdmin, "", broker.Event{
			Type:      "job.status_changed",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"booking_id": bookingID,
				"status":     status,
				"tech_id":    booking.TechnicianID,
			},
		})
	}

	// Notify Tech (FCM)
	if s.notifications != nil && booking.TechnicianID != "" {
		go func() {
			tech, err := s.techRepo.GetByID(booking.TechnicianID)
			if err == nil && tech.FCMToken != "" {
				err := s.notifications.NotifyJobStatusChange(context.Background(), tech.FCMToken, bookingID, status)
				if err != nil && err.Error() == "token_invalid" {
					log.Printf("🧹 [BOOKING_SERVICE] Cleaning up invalid token for tech %s in UpdateStatus", tech.ID)
					tech.FCMToken = ""
					if err := s.techRepo.Update(tech); err != nil {
						log.Printf("⚠️ [BOOKING_SERVICE] Failed to clear invalid token: %v", err)
					}
				}
			}
		}()
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
	// [FIX] If booking has no coordinates (Lat/Long = 0), we assume the technician is at the correct location.
	// We Update the booking's location to the technician's current location to "fix" the data.
	if booking.Lat == 0 || booking.Long == 0 {
		log.Printf("⚠️ Booking %s has no coordinates. Auto-updating to tech location: %.6f, %.6f", bookingID, techLat, techLong)
		booking.Lat = techLat
		booking.Long = techLong
		// We don't save yet, we save at the end of function along with status update.
		// Distance check is skipped (effectively dist=0)
	} else {
		// Normal Check
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

	// [NEW] FCM to Admin (Multicast) AND Tech
	if s.notifications != nil {
		go func() {
			// 1. Notify Admins
			settings, err := s.settingsRepo.GetSettings()
			if err == nil && len(settings.AdminFCMTokens) > 0 {
				failedTokens, err := s.notifications.NotifyAdminsBookingCancelled(context.Background(), settings.AdminFCMTokens, bookingID, booking.CustomerName, reason, note)
				if err != nil {
					log.Printf("❌ [BOOKING_SERVICE] Failed to notify admins of cancellation: %v", err)
				}
				// Cleanup invalid tokens
				if len(failedTokens) > 0 {
					for _, t := range failedTokens {
						_ = s.settingsRepo.RemoveAdminToken(t)
					}
				}
			}

			// 2. Notify Tech (if assigned)
			if booking.TechnicianID != "" {
				tech, err := s.techRepo.GetByID(booking.TechnicianID)
				if err == nil && tech.FCMToken != "" {
					// Re-use NotifyJobStatusChange or create specific one.
					// Since "cancelled" is a status, NotifyJobStatusChange should work if it handles "cancelled".
					// Let's check s.notifications.NotifyJobStatusChange implementation.
					// It maps "pending", "assigned", "in_progress", "completed".
					// Does it map "cancelled"? lines 203-208 in fcm_service.go
					// It does NOT listed "cancelled" explicitly in the map, but defaults to "Cập nhật trạng thái...".
					// Let's use NotifyJobStatusChange and assume it handles it or generic fallback is fine.
					// Re-use NotifyJobStatusChange or create specific one.
					err := s.notifications.NotifyJobStatusChange(context.Background(), tech.FCMToken, bookingID, "cancelled")
					if err != nil && err.Error() == "token_invalid" {
						log.Printf("🧹 [BOOKING_SERVICE] Cleaning up invalid token for tech %s", tech.ID)
						tech.FCMToken = ""
						if err := s.techRepo.Update(tech); err != nil {
							log.Printf("⚠️ [BOOKING_SERVICE] Failed to clear invalid token: %v", err)
						}
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
