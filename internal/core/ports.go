package core

import (
	"context"

	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// BookingRepository defines data access methods for Bookings
type BookingRepository interface {
	GetByID(id string) (*Booking, error)
	Create(booking *Booking, files []*filesystem.File) error
	Update(booking *Booking) error

	// Specialized queries
	FindPending() ([]*Booking, error)
	FindActiveByTechnician(techID string) ([]*Booking, error)    // Active = Assigned & In Progress (not completed)
	FindScheduledByTechnician(techID string) ([]*Booking, error) // Scheduled = Not Cancelled (includes completed)
}

// TechnicianRepository defines data access for Technicians
type TechnicianRepository interface {
	GetByID(id string) (*Technician, error)
	GetAvailable() ([]*Technician, error)
	GetAll() ([]*Technician, error)
	Create(tech *Technician, password string) error
	Update(tech *Technician) error
	SetPassword(id, password string) error
	ToggleActive(id string) error
}

type TimeSlotRepository interface {
	GetByID(id string) (*TimeSlot, error)
	Update(slot *TimeSlot) error
	// FindByDate could be added later
}

type ServiceRepository interface {
	GetByID(id string) (*Service, error)
}

// TimeSlotControl defines business logic for time slots
type TimeSlotControl interface {
	ReleaseSlot(slotID string) error
	BookSlot(slotID, bookingID string) error
	CheckConflict(techID, date, timeStr string, durationMin int) error
}

type AnalyticsRepository interface {
	GetDailyRevenue(start, end string) ([]RevenueStat, error)
	GetTopTechnicians(limit int) ([]TechPerformance, error)
	GetTotalRevenue() (float64, error)
	CountBookings(filter string) (int64, error)
	CountTechnicians(filter string) (int64, error)
}

type AnalyticsService interface {
	GetRevenueLast7Days() ([]RevenueStat, error)
	GetTopTechnicians(limit int) ([]TechPerformance, error)
	GetDashboardStats() (*DashboardStats, error)
}

type NotificationService interface {
	NotifyNewJobAssignment(ctx context.Context, techToken string, jobID string, customerName string) error
}

// BookingService defines business logic methods
type BookingService interface {
	CreateBooking(req *BookingRequest) (*Booking, error)
	AssignTechnician(bookingID, technicianID string) error
	RecallToPending(bookingID string) error
	UpdateStatus(bookingID, status string) error
	TechCheckIn(bookingID string, techLat, techLong float64) error
	CancelBooking(bookingID, reason, note string) error
	RescheduleBooking(bookingID, newTime string) error
}

// DTOs for Service Layer
type BookingRequest struct {
	ServiceID      string
	CustomerName   string
	Phone          string
	Address        string
	AddressDetails string
	IssueDesc      string
	DeviceType     string
	Brand          string
	BookingTime    string
	SlotID         string
	Lat            float64
	Long           float64
	Files          []*filesystem.File
}
