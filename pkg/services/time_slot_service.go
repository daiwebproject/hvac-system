package services

import (
	"fmt"
	"strings"
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
	Status          string // available, limited, waitlist, full
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
	var unassignedBlocks []TimeBlock // [Fix] Track unassigned bookings

	fmt.Printf("[DEBUG] Date: %s | Total Bookings Found: %d\n", date, len(bookings))

	for _, b := range bookings {
		techID := b.TechnicianID
		// if techID == "" { continue } // REMOVED: Don't skip unassigned

		startTimeStr := b.BookingTime // "2006-01-02 15:04"
		// Try parsing with flexible formats
		var startT time.Time
		var err error

		formats := []string{"2006-01-02 15:04", "2006-01-02 15:04:05", "2006-01-02 15:04:05.000Z", time.RFC3339}
		parsed := false
		for _, f := range formats {
			startT, err = time.Parse(f, startTimeStr)
			if err == nil {
				parsed = true
				break
			}
		}

		if parsed {
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

			if techID != "" {
				techSchedules[techID] = append(techSchedules[techID], TimeBlock{
					Start: startT,
					End:   endT,
				})
			} else {
				// Add to unassigned pool
				unassignedBlocks = append(unassignedBlocks, TimeBlock{
					Start: startT,
					End:   endT,
				})
				fmt.Printf("[DEBUG] Unassigned Block Added: %s - %s\n", startT.Format("15:04"), endT.Format("15:04"))
			}
		} else {
			fmt.Printf("[DEBUG] Failed to parse booking time: '%s'\n", startTimeStr)
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

		// [Fix] Subtract unassigned bookings that overlap with this slot
		unassignedOverlaps := 0
		for _, block := range unassignedBlocks {
			// Slot [Start, End] overlaps Block [Start, End]?
			if slotStart.Before(block.End) && slotEnd.After(block.Start) {
				unassignedOverlaps++
			}
		}

		// Net Availability
		availableTechsCount = availableTechsCount - unassignedOverlaps
		if availableTechsCount < 0 {
			availableTechsCount = 0
		}

		// 6. Determine Status & Availability
		// available: > 2
		// limited: 1-2
		// waitlist: 0 (but allow +2 overbooking)
		// full: 0 (and waitlist full)

		status := "full"
		isAvailable := false

		// Occupied slots (approximate based on dynamic calculation)
		currentOccupancy := dynamicCapacity - availableTechsCount

		if availableTechsCount >= 2 {
			status = "available"
			isAvailable = true
		} else if availableTechsCount > 0 {
			status = "limited"
			isAvailable = true
		} else {
			// Waitlist logic: Allow 2 extra bookings beyond capacity
			// We need to check exact booking count for this slot to see if waitlist is full
			// For now, let's approximate: Check strictly assigned jobs + 2
			// But we don't track "Waitlist" in DB yet explicitly beyond booking records.
			// Getting total bookings for this slot specifically from DB would be accurate.
			// Using 'record.CurrentBookings' from 'time_slots' collection is the persistent counter.
			persistentBookings := int(record.GetFloat("current_bookings"))

			if persistentBookings < dynamicCapacity+2 {
				status = "waitlist"
				isAvailable = true
			} else {
				status = "full"
				isAvailable = false
			}
		}

		slots = append(slots, TimeSlot{
			ID:              record.Id,
			Date:            record.GetString("date"),
			StartTime:       startTimeStr,
			EndTime:         record.GetString("end_time"),
			MaxCapacity:     dynamicCapacity,
			CurrentBookings: currentOccupancy,
			IsAvailable:     isAvailable,
			Status:          status,
		})
	}

	return slots, nil
}

// GetAvailableSlotsWithFilters returns available time slots filtered by customer zone and service skill
// This enables "Smart Booking" - only show slots where qualified techs are available
func (s *TimeSlotService) GetAvailableSlotsWithFilters(date string, customerZone string, serviceID string) ([]TimeSlot, error) {
	const StandardSlotDuration = 120 * time.Minute
	const TravelBuffer = 30 * time.Minute

	// Validate date
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date format: %w", err)
	}
	if targetDate.Before(time.Now().Truncate(24 * time.Hour)) {
		return []TimeSlot{}, nil
	}

	// Get required skill from service (if any)
	requiredSkill := ""
	if serviceID != "" {
		service, err := s.app.FindRecordById("services", serviceID)
		if err == nil && service != nil {
			requiredSkill = service.GetString("required_skill")
		}
	}

	// Get all active techs then filter by zone/skill
	allTechs, err := s.techRepo.GetAvailable()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch technicians: %w", err)
	}

	// Filter techs by criteria
	eligibleTechs := []*domain.Technician{}
	for _, tech := range allTechs {
		// Check skill requirement
		if requiredSkill != "" && !techHasSkill(tech.Skills, requiredSkill) {
			fmt.Printf("[SMART_BOOKING] Tech %s skipped: missing skill '%s'\n", tech.Name, requiredSkill)
			continue
		}

		// Check zone coverage
		if customerZone != "" && !techCoversZone(tech.ServiceZones, customerZone) {
			fmt.Printf("[SMART_BOOKING] Tech %s skipped: zone '%s' not covered\n", tech.Name, customerZone)
			continue
		}

		eligibleTechs = append(eligibleTechs, tech)
	}

	dynamicCapacity := len(eligibleTechs)
	fmt.Printf("[SMART_BOOKING] Date=%s | Eligible Techs: %d/%d (Zone=%s, Skill=%s)\n",
		date, dynamicCapacity, len(allTechs), customerZone, requiredSkill)

	if dynamicCapacity == 0 {
		return []TimeSlot{}, nil
	}

	// Build schedules for eligible techs only
	bookings, _ := s.bookingRepo.FindAllByDate(date)
	techSchedules := make(map[string][]TimeBlock)
	var unassignedBlocks []TimeBlock

	for _, b := range bookings {
		startT, err := time.Parse("2006-01-02 15:04", b.BookingTime)
		if err != nil {
			continue
		}
		duration := 60 * time.Minute
		endT := startT.Add(duration).Add(TravelBuffer)

		if b.TechnicianID != "" {
			techSchedules[b.TechnicianID] = append(techSchedules[b.TechnicianID], TimeBlock{Start: startT, End: endT})
		} else {
			unassignedBlocks = append(unassignedBlocks, TimeBlock{Start: startT, End: endT})
		}
	}

	// Fetch slot definitions
	filter := fmt.Sprintf("date = '%s'", date)
	records, err := s.app.FindRecordsByFilter("time_slots", filter, "start_time", 100, 0, nil)
	if err != nil {
		return nil, err
	}

	// Auto-generate if needed
	if len(records) == 0 {
		if err := s.GenerateDefaultSlots(date, dynamicCapacity); err == nil {
			records, _ = s.app.FindRecordsByFilter("time_slots", filter, "start_time", 100, 0, nil)
		}
	}

	slots := make([]TimeSlot, 0, len(records))
	now := time.Now()

	for _, record := range records {
		startTimeStr := record.GetString("start_time")
		slotStart, _ := time.Parse("2006-01-02 15:04", date+" "+startTimeStr)
		slotEnd := slotStart.Add(StandardSlotDuration)

		// Skip past slots
		if date == now.Format("2006-01-02") && slotStart.Sub(now).Hours() < 2 {
			continue
		}

		// Count eligible free techs
		availableTechsCount := 0
		for _, tech := range eligibleTechs {
			timeline := techSchedules[tech.ID]
			isFree := true
			for _, block := range timeline {
				if slotStart.Before(block.End) && slotEnd.After(block.Start) {
					isFree = false
					break
				}
			}
			if isFree {
				availableTechsCount++
			}
		}

		// Subtract unassigned overlaps
		for _, block := range unassignedBlocks {
			if slotStart.Before(block.End) && slotEnd.After(block.Start) {
				availableTechsCount--
			}
		}
		if availableTechsCount < 0 {
			availableTechsCount = 0
		}

		// Determine status
		status := "full"
		isAvailable := false
		if availableTechsCount >= 2 {
			status = "available"
			isAvailable = true
		} else if availableTechsCount > 0 {
			status = "limited"
			isAvailable = true
		} else {
			persistentBookings := int(record.GetFloat("current_bookings"))
			if persistentBookings < dynamicCapacity+2 {
				status = "waitlist"
				isAvailable = true
			}
		}

		slots = append(slots, TimeSlot{
			ID:              record.Id,
			Date:            record.GetString("date"),
			StartTime:       startTimeStr,
			EndTime:         record.GetString("end_time"),
			MaxCapacity:     dynamicCapacity,
			CurrentBookings: dynamicCapacity - availableTechsCount,
			IsAvailable:     isAvailable,
			Status:          status,
		})
	}

	return slots, nil
}

// techHasSkill checks if technician has the required skill
func techHasSkill(techSkills []string, requiredSkill string) bool {
	for _, skill := range techSkills {
		if skill == requiredSkill {
			return true
		}
	}
	return false
}

// techCoversZone checks if technician covers the customer's zone
// Matching logic: exact match OR partial match (district/province level)
func techCoversZone(techZones []string, customerZone string) bool {
	if len(techZones) == 0 {
		return true // No zone restriction = covers all
	}
	for _, zone := range techZones {
		// Exact match
		if zone == customerZone {
			return true
		}
		// Partial match: if both contain same district or province
		// Zone format: "Xã ABC, Huyện XYZ, Tỉnh DEF"
		techParts := strings.Split(zone, ", ")
		custParts := strings.Split(customerZone, ", ")

		// Match at district level (2nd part) or province level (3rd part)
		if len(techParts) >= 2 && len(custParts) >= 2 {
			if techParts[1] == custParts[1] { // Same district
				return true
			}
		}
		if len(techParts) >= 3 && len(custParts) >= 3 {
			if techParts[2] == custParts[2] { // Same province
				return true
			}
		}
	}
	return false
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
// CheckConflict validates if a specific scheduler slot conflicts with existing bookings for a technician
func (s *TimeSlotService) CheckConflict(techID string, date string, startTime string, newJobDuration int, newSlotID string, excludeBookingID string) error {
	const TravelBufferMinutes = 30

	// 1. Tính toán thời gian của Job MỚI đang định giao
	newStart, err := time.Parse("2006-01-02 15:04", date+" "+startTime)
	if err != nil {
		return fmt.Errorf("invalid date format: %v", err)
	}
	// Thời gian kết thúc = Bắt đầu + Thời lượng dịch vụ
	newEnd := newStart.Add(time.Duration(newJobDuration) * time.Minute)

	// 2. Lấy danh sách các Job ĐÃ CÓ của thợ trong ngày đó
	// Lọc các job chưa huỷ (cancelled) và CHƯA HOÀN THÀNH (completed).
	// Nếu job đã xong, thợ coi như rảnh (hoặc chấp nhận overlap vì đã xong việc).
	records, err := s.app.FindRecordsByFilter(
		"bookings",
		fmt.Sprintf("technician_id='%s' && job_status != 'cancelled' && job_status != 'completed'", techID),
		"booking_time",
		100,
		0,
		nil,
	)
	if err != nil { // CheckConflict
		return fmt.Errorf("failed to fetch existing jobs: %w", err)
	}

	for _, job := range records {
		// [FIX] Skip the current booking if updating/re-assigning
		if excludeBookingID != "" && job.Id == excludeBookingID {
			continue
		}

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
