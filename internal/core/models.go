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
	FCMToken string `json:"fcm_token"` // [NEW]
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

// ============ REAL-TIME TRACKING MODELS ============

// LocationUpdate represents a real-time location update from technician
type LocationUpdate struct {
	TechnicianID string    `json:"technician_id"`
	BookingID    string    `json:"booking_id"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Accuracy     float64   `json:"accuracy"`
	Timestamp    int64     `json:"timestamp"`
	Speed        float64   `json:"speed,omitempty"`
	Heading      float64   `json:"heading,omitempty"`
}

// TechStatus represents current technician's location and status
type TechStatus struct {
	TechnicianID   string    `json:"technician_id"`
	TechnicianName string    `json:"technician_name"`
	CurrentBooking string    `json:"current_booking"`
	Status         string    `json:"status"` // idle, moving, arrived, working, completed
	Latitude       float64   `json:"latitude"`
	Longitude      float64   `json:"longitude"`
	LastUpdate     int64     `json:"last_update"`
	Distance       float64   `json:"distance,omitempty"` // Distance to customer location in meters
}

// GeofenceEvent represents a geofencing alert
type GeofenceEvent struct {
	Type         string `json:"type"` // arrived, departed, geofence_enter
	TechnicianID string `json:"technician_id"`
	BookingID    string `json:"booking_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Distance     float64 `json:"distance"`
	Timestamp    int64  `json:"timestamp"`
}
