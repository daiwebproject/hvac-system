package core

// Booking represents a service request from a customer
type Booking struct {
	ID               string `json:"id"`
	ServiceID        string `json:"service_id"`
	CustomerName     string `json:"customer_name"`
	CustomerPhone    string `json:"customer_phone"`
	Address          string `json:"address"`
	AddressDetails   string `json:"address_details"`
	IssueDescription string `json:"issue_description"`
	DeviceType       string `json:"device_type"`
	Brand            string `json:"brand"`
	JobStatus        string `json:"job_status"`
	BookingTime      string `json:"booking_time"` // Raw string from DB for now, ideally time.Time

	// Assignments
	TechnicianID string  `json:"technician_id"`
	SlotID       *string `json:"slot_id"` // Pointer to allow null

	// Timestamps (using string for simplicity in transition, can be time.Time)
	Created string `json:"created"`
	Updated string `json:"updated"`

	// Coordinates
	Lat  float64 `json:"lat"`
	Long float64 `json:"long"`

	// Status timestamps
	ArrivedAt string `json:"arrived_at"`

	// Exception Handling
	CancelReason string `json:"cancel_reason"`
}

// Technician represents a service staff member
type Technician struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Active   bool   `json:"active"`
	Verified bool   `json:"verified"`
}

// TimeSlot represents a bookable time window
type TimeSlot struct {
	ID              string `json:"id"`
	Date            string `json:"date"`
	StartTime       string `json:"start_time"`
	EndTime         string `json:"end_time"`
	MaxCapacity     int    `json:"max_capacity"`
	CurrentBookings int    `json:"current_bookings"`
	IsAvailable     bool   `json:"is_available"`
	IsBooked        bool   `json:"is_booked"`
}

// Service represents a service offering
type Service struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	DurationMinutes int    `json:"duration_minutes"`
}

// Analytics Models
type RevenueStat struct {
	Date   string  `json:"date"`
	Amount float64 `json:"amount"` // Daily Total
}

type TechPerformance struct {
	TechnicianID   string  `json:"technician_id"`
	TechnicianName string  `json:"technician_name"`
	CompletedJobs  int     `json:"completed_jobs"`
	TotalRevenue   float64 `json:"total_revenue"`
}

type DashboardStats struct {
	TotalRevenue   float64
	BookingsToday  int
	ActiveTechs    int
	PendingCount   int
	CompletedCount int
	CompletionRate float64
}
