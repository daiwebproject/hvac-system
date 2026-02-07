package services

import (
	"fmt"
	"time"

	domain "hvac-system/internal/core"

	"github.com/pocketbase/pocketbase/core"
)

// TimeSlotService handles time slot availability and booking logic
type TimeSlotService struct {
	app         core.App
	techRepo    domain.TechnicianRepository
	bookingRepo domain.BookingRepository
}

// NewTimeSlotService creates a new time slot service
func NewTimeSlotService(app core.App, techRepo domain.TechnicianRepository, bookingRepo domain.BookingRepository) *TimeSlotService {
	return &TimeSlotService{
		app:         app,
		techRepo:    techRepo,
		bookingRepo: bookingRepo,
	}
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

// TimeBlock represents a busy period for a technician
type TimeBlock struct {
	Start time.Time
	End   time.Time
}

// GetAvailableSlots returns available time slots using Dynamic Availability
// Formula: Available = TotalActiveTechs - (AssignedJobs + StuckJobs)
func (s *TimeSlotService) GetAvailableSlots(date string) ([]TimeSlot, error) {
	// Standard slot duration (could be dynamic based on selected service in future)
	const StandardSlotDuration = 120 * time.Minute // 2 hours
	const TravelBuffer = 30 * time.Minute

	// Validate date format
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}

	// Don't allow booking in the past
	if targetDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return []TimeSlot{}, nil
	}

	// 1. Get Resources (Total Active Technicians)
	dynamicCapacity, err := s.techRepo.CountActive()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch active technicians: %w", err)
	}
	if dynamicCapacity == 0 {
		return []TimeSlot{}, nil // No techs = no slots
	}

	// 2. Get Constraints (All Bookings for the Date)
	bookings, err := s.bookingRepo.FindAllByDate(date)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch bookings: %w", err)
	}

	// 3. Build Tech Timelines (Map[TechID] -> BusyBlocks)
	techSchedules := make(map[string][]TimeBlock)
	for _, b := range bookings {
		techID := b.TechnicianID
		if techID == "" {
			continue // Unassigned jobs don't block dynamic capacity yet
		}

		startTimeStr := b.BookingTime // "2006-01-02 15:04"
		if startT, err := time.Parse("2006-01-02 15:04", startTimeStr); err == nil {
			// Estimate duration (default 60m + travel 30m)
			// Ideally we fetch service specific duration, for now allow standard block
			duration := 60 * time.Minute

			// [TODO] If ServiceID is available, fetch duration
			// For MVP, we use standard block or existing logic if available

			endT := startT.Add(duration).Add(TravelBuffer)

			// [NEW] Overrun Detection (Stuck Job)
			if b.JobStatus == "working" && endT.Before(time.Now()) {
				estimatedEnd := time.Now().Add(30 * time.Minute)
				if estimatedEnd.After(endT) {
					endT = estimatedEnd
				}
			}

			techSchedules[techID] = append(techSchedules[techID], TimeBlock{
				Start: startT,
				End:   endT,
			})
		}
	}

	// 4. Fetch Standard Slot Definitions (The "Grid")
	// We still use the "time_slots" collection to define the grid (8-10, 10-12...)
	filter := fmt.Sprintf("date = '%s'", date)
	records, err := s.app.FindRecordsByFilter("time_slots", filter, "start_time", 100, 0, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch slot definitions: %w", err)
	}

	// [AUTO-GENERATE] If no slots exist for this date, create them on-the-fly
	if len(records) == 0 {
		// Only auto-generate for future dates (not past)
		if !targetDate.Before(time.Now().Truncate(24 * time.Hour)) {
			if err := s.GenerateDefaultSlots(date, dynamicCapacity); err == nil {
				// Re-fetch after generation
				records, _ = s.app.FindRecordsByFilter("time_slots", filter, "start_time", 100, 0, nil)
			}
		}
	}

	slots := make([]TimeSlot, 0, len(records))
	now := time.Now()

	for _, record := range records {
		startTimeStr := record.GetString("start_time")
		slotStart, _ := time.Parse("2006-01-02 15:04", date+" "+startTimeStr)
		slotEnd := slotStart.Add(StandardSlotDuration)

		// Filter past slots (2h advance notice)
		if date == now.Format("2006-01-02") {
			if slotStart.Sub(now).Hours() < 2 {
				continue
			}
		}

		// 5. Evaluate Availability: Count how many techs are free for this slot
		availableTechsCount := 0

		// Use CountActive() implicitly by assuming specific Tech IDs aren't needed here,
		// but we do need to iterate actual Tech IDs if we built specific schedules.
		// Wait, CountActive returns int. We need the list of Active Techs to check their specific schedules?
		// YES.
		// Retrying fetching active techs list for iteration
		// Optimally this should be cached or passed from handler, but repo query is fast enough (sqlite/pb)
		activeTechs, _ := s.techRepo.GetAvailable() // Re-using GetAvailable which returns []*Technician

		for _, tech := range activeTechs {
			techID := tech.ID
			timeline := techSchedules[techID]
			isFree := true

			// Check intersection with any busy block
			for _, block := range timeline {
				// Slot [Start, End] overlaps Block [Start, End]?
				// Allow TravelBuffer logic inside Block definition
				if slotStart.Before(block.End) && slotEnd.After(block.Start) {
					isFree = false
					break
				}
			}

			if isFree {
				availableTechsCount++
			}
		}

		slots = append(slots, TimeSlot{
			ID:              record.Id,
			Date:            record.GetString("date"),
			StartTime:       startTimeStr,
			EndTime:         record.GetString("end_time"),
			MaxCapacity:     dynamicCapacity,
			CurrentBookings: dynamicCapacity - availableTechsCount, // Occupied = Total - Free
			IsAvailable:     availableTechsCount > 0,
		})
	}

	return slots, nil
}

// BookSlot reserves a time slot (Legacy Wrapper)
// In the new logic, we don't strictly "book" a slot ID counters,
// but we perform a validation check.
func (s *TimeSlotService) BookSlot(slotID, bookingID string) error {
	slot, err := s.app.FindRecordById("time_slots", slotID)
	if err != nil {
		return fmt.Errorf("slot not found")
	}

	// Double-check availability logic
	date := slot.GetString("date")

	// Re-run smart check just for this slot?
	// For performance, we might trust the generic capacity check for now,
	// or implement a lightweight "IsSlotAvailable(date, start, duration)" helper.

	// MVP: Fetch availability for the whole day (cached?) and check this slot.
	availableSlots, err := s.GetAvailableSlots(date)
	if err != nil {
		return err
	}

	isAvailable := false
	for _, s := range availableSlots {
		if s.ID == slotID {
			if s.IsAvailable {
				isAvailable = true
			}
			break
		}
	}

	if !isAvailable {
		return fmt.Errorf("slot is no longer available")
	}

	// We can update `current_bookings` just for analytics/counters
	// But it is no longer the Source of Truth for availability.
	currentBookings := slot.GetFloat("current_bookings")
	slot.Set("current_bookings", currentBookings+1)
	s.app.Save(slot)

	return nil
}

// CheckConflict validates if a specific scheduler slot conflicts with existing bookings for a technician
// Rules:
// 1. New Job Start Time >= Previous Job End Time + Travel Buffer (30m)
// 2. New Job End Time + Travel Buffer <= Next Job Start Time
// CheckConflict validates if a specific scheduler slot conflicts with existing bookings
func (s *TimeSlotService) CheckConflict(techID string, date string, startTime string, newJobDuration int, newSlotID string) error {
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

		// [NEW] Check Slot overlap if both have slot IDs
		if newSlotID != "" {
			existingSlot := job.GetString("time_slot_id")
			if existingSlot == newSlotID {
				return fmt.Errorf("Xung đột: Thợ đã được giao việc trong khung giờ này (Trùng Slot ID)")
			}
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
		// Quy tắc:
		// - Job Mới phải bắt đầu SAU khi Job Cũ kết thúc + 30p
		// - Job Mới phải kết thúc TRƯỚC khi Job Cũ bắt đầu - 30p (nếu chen vào trước)

		// Thời gian an toàn mà Job Cũ chiếm dụng (bao gồm cả di chuyển đến và đi)
		// [Start - 30] ... [End + 30]
		// Nếu Job Mới chạm vào khoảng này thì là Conflict

		bufferedStart := jobStart.Add(-time.Duration(TravelBufferMinutes) * time.Minute)
		bufferedEnd := jobEnd.Add(time.Duration(TravelBufferMinutes) * time.Minute)

		if newEnd.After(bufferedStart) && newStart.Before(bufferedEnd) {
			return fmt.Errorf("Xung đột lịch trình: Thợ đã có việc từ %s đến %s (cộng thời gian di chuyển)",
				jobStart.Format("15:04"), jobEnd.Format("15:04"))
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
