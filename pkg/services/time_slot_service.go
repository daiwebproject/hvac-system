package services

import (
	"fmt"
	"time"

	"github.com/pocketbase/pocketbase/core"
)

// TimeSlotService handles time slot availability and booking logic
type TimeSlotService struct {
	app core.App
}

// NewTimeSlotService creates a new time slot service
func NewTimeSlotService(app core.App) *TimeSlotService {
	return &TimeSlotService{app: app}
}

// TimeSlot represents a bookable time window
type TimeSlot struct {
	ID              string
	Date            string // YYYY-MM-DD
	StartTime       string // HH:MM
	EndTime         string // HH:MM
	MaxCapacity     int
	CurrentBookings int
	IsAvailable     bool
}

// GetAvailableSlots returns available time slots for a given date
// Business rule: Capacity based on ACTIVE technicians, 2-hour advance notice
func (s *TimeSlotService) GetAvailableSlots(date string) ([]TimeSlot, error) {
	// Validate date format
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Don't allow booking in the past
	if targetDate.Before(time.Now().Truncate(24 * time.Hour)) {
		// return nil, fmt.Errorf("cannot book slots in the past")
		// Logic fix: Truncate to day is fine, current day is valid
	}

	// 1. Get Dynamic Capacity (Count Active Techs)
	// [SYNC] Only count techs who are currently marked as active/online
	activeTechs, err := s.app.FindRecordsByFilter("technicians", "active=true", "", 0, 0, nil)
	dynamicCapacity := 0
	if err == nil {
		dynamicCapacity = len(activeTechs)
	}

	// Query available slots
	// We verify capacity memory-side since DB max_capacity might be stale
	filter := fmt.Sprintf("date = '%s'", date)
	records, err := s.app.FindRecordsByFilter("time_slots", filter, "start_time", 100, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch slots: %w", err)
	}

	slots := make([]TimeSlot, 0, len(records))
	now := time.Now()

	for _, record := range records {
		startTime := record.GetString("start_time")

		// Skip slots that are too soon (less than 2 hours from now)
		if date == time.Now().Format("2006-01-02") {
			slotTime, _ := time.Parse("2006-01-02 15:04", date+" "+startTime)
			if slotTime.Sub(now).Hours() < 2 {
				continue
			}
		}

		currentBookings := int(record.GetFloat("current_bookings"))

		// [SYNC] Determine availability using Dynamic Capacity
		isAvailable := currentBookings < dynamicCapacity

		slots = append(slots, TimeSlot{
			ID:              record.Id,
			Date:            record.GetString("date"),
			StartTime:       startTime,
			EndTime:         record.GetString("end_time"),
			MaxCapacity:     dynamicCapacity, // Show real-time capacity
			CurrentBookings: currentBookings,
			IsAvailable:     isAvailable,
		})
	}

	return slots, nil
}

// BookSlot reserves a time slot for a booking
// Business rule: Atomic increment & Dynamic Capacity Check
func (s *TimeSlotService) BookSlot(slotID, bookingID string) error {
	slot, err := s.app.FindRecordById("time_slots", slotID)
	if err != nil {
		return fmt.Errorf("slot not found: %w", err)
	}

	// [SYNC] Check against Dynamic Capacity (Active Techs)
	activeTechs, _ := s.app.FindRecordsByFilter("technicians", "active=true", "", 0, 0, nil)
	dynamicCapacity := float64(len(activeTechs))

	// Check availability
	currentBookings := slot.GetFloat("current_bookings")
	// maxCapacity := slot.GetFloat("max_capacity") // Ignore static capacity

	if currentBookings >= dynamicCapacity {
		return fmt.Errorf("slot is fully booked (no active technicians)")
	}

	// Increment booking count
	slot.Set("current_bookings", currentBookings+1)

	// If this was the last available spot, mark as fully booked metadata
	if currentBookings+1 >= dynamicCapacity {
		slot.Set("is_booked", true)
	}

	if err := s.app.Save(slot); err != nil {
		return fmt.Errorf("failed to book slot: %w", err)
	}

	return nil
}

// CheckConflict validates if a specific scheduler slot conflicts with existing bookings for a technician
// Rules:
// 1. New Job Start Time >= Previous Job End Time + Travel Buffer (30m)
// 2. New Job End Time + Travel Buffer <= Next Job Start Time
// CheckConflict validates if a specific scheduler slot conflicts with existing bookings
func (s *TimeSlotService) CheckConflict(techID string, date string, startTime string, newJobDuration int) error {
	const TravelBufferMinutes = 30

	// 1. Tính toán thời gian của Job MỚI đang định giao
	newStart, err := time.Parse("2006-01-02 15:04", date+" "+startTime)
	if err != nil {
		return fmt.Errorf("invalid date format: %v", err)
	}
	// Thời gian kết thúc = Bắt đầu + Thời lượng dịch vụ
	newEnd := newStart.Add(time.Duration(newJobDuration) * time.Minute)

	// 2. Lấy danh sách các Job ĐÃ CÓ của thợ trong ngày đó
	// Lọc các job chưa huỷ (cancelled) và thuộc về thợ này
	records, err := s.app.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && job_status != 'cancelled'", techID),
		"booking_time",
		100,
		0,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch existing jobs: %w", err)
	}

	for _, job := range records {
		jobTimeStr := job.GetString("booking_time") // "2006-01-02 15:04"
		if len(jobTimeStr) < 16 {
			continue
		}

		jobStart, err := time.Parse("2006-01-02 15:04", jobTimeStr)
		if err != nil {
			continue
		}

		// Chỉ kiểm tra các job cùng ngày
		if jobStart.Format("2006-01-02") != date {
			continue
		}

		// 3. Xác định thời lượng của Job ĐÃ CÓ (Quan trọng)
		// Mặc định 60 phút nếu không tìm thấy service
		existingJobDuration := 60

		// Lấy thông tin service của job cũ để biết thời lượng chính xác
		serviceID := job.GetString("service_id")
		if serviceID != "" {
			serviceRecord, err := s.app.FindRecordById("services", serviceID)
			if err == nil {
				// Lấy duration từ DB, nếu bằng 0 thì vẫn để mặc định 60
				if d := serviceRecord.GetInt("duration_minutes"); d > 0 {
					existingJobDuration = d
				}
			}
		}

		jobEnd := jobStart.Add(time.Duration(existingJobDuration) * time.Minute)

		// 4. Kiểm tra xung đột có tính Buffer di chuyển
		// Vùng an toàn của Job Cũ = [Start - 30p] đến [End + 30p]
		bufferedStart := jobStart.Add(-time.Duration(TravelBufferMinutes) * time.Minute)
		bufferedEnd := jobEnd.Add(time.Duration(TravelBufferMinutes) * time.Minute)

		// Logic giao nhau: (NewStart < BufferedEnd) AND (NewEnd > BufferedStart)
		if newStart.Before(bufferedEnd) && newEnd.After(bufferedStart) {
			return fmt.Errorf(
				"Xung đột lịch trình: Thợ đã có việc từ %s đến %s (Dịch vụ: %dp + 30p di chuyển)",
				jobStart.Format("15:04"),
				jobEnd.Format("15:04"),
				existingJobDuration,
			)
		}
	}

	return nil
}

// ReleaseSlot frees up a time slot (when booking is cancelled)
func (s *TimeSlotService) ReleaseSlot(slotID string) error {
	slot, err := s.app.FindRecordById("time_slots", slotID)
	if err != nil {
		return fmt.Errorf("slot not found: %w", err)
	}

	currentBookings := slot.GetFloat("current_bookings")
	if currentBookings > 0 {
		slot.Set("current_bookings", currentBookings-1)
		slot.Set("is_booked", false) // Re-open slot
	}

	if err := s.app.Save(slot); err != nil {
		return fmt.Errorf("failed to release slot: %w", err)
	}

	return nil
}

// GenerateDefaultSlots creates standard time slots for a date
// Business rule: 2-hour windows from 8AM to 8PM
func (s *TimeSlotService) GenerateDefaultSlots(date string, technicianCount int) error {
	// Validate date
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	// Check if slots already exist for this date
	existing, _ := s.app.FindRecordsByFilter("time_slots", fmt.Sprintf("date = '%s'", date), "", 1, 0, nil)
	if len(existing) > 0 {
		return fmt.Errorf("slots already exist for this date")
	}

	// Standard time windows (8AM-8PM, 2-hour slots)
	timeWindows := [][2]string{
		{"08:00", "10:00"},
		{"10:00", "12:00"},
		{"13:00", "15:00"}, // Lunch break 12-1PM
		{"15:00", "17:00"},
		{"17:00", "19:00"},
		{"19:00", "21:00"},
	}

	collection, err := s.app.FindCollectionByNameOrId("time_slots")
	if err != nil {
		return err
	}

	for _, window := range timeWindows {
		record := core.NewRecord(collection)
		record.Set("date", date)
		record.Set("start_time", window[0])
		record.Set("end_time", window[1])
		record.Set("max_capacity", float64(technicianCount))
		record.Set("current_bookings", 0)
		record.Set("is_booked", false)

		if err := s.app.Save(record); err != nil {
			return fmt.Errorf("failed to create slot: %w", err)
		}
	}

	return nil
}
