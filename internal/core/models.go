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
	TechNotes    string `json:"tech_notes"` // [NEW] Technician's private notes

	// UI Helpers
	StatusOrder int `json:"status_order"` // 1:accepted, 2:moving, 3:arrived, 4:working, 5:completed
}

// Technician represents a service staff member
type Technician struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
	Active   bool   `json:"active"`
	Verified bool   `json:"verified"`
	FCMToken string `json:"fcm_token"`

	// [UPGRADE] Professional Management Fields
	Phone          string   `json:"phone"`           // Contact number
	Rating         float64  `json:"rating"`          // Average Rating (0-5)
	Level          string   `json:"level"`           // Junior, Senior, Master
	ServiceZones   []string `json:"service_zones"`   // JSON Array of Zone IDs
	Skills         []string `json:"skills"`          // JSON Array of Service/Skill IDs
	SkillNames     []string `json:"skill_names"`     // [Display] Resolved Names
	BaseSalary     float64  `json:"base_salary"`     // Monthly base salary
	CommissionRate float64  `json:"commission_rate"` // Personal override rate (%)
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

// Category represents a grouping of services
type Category struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ParentID  string `json:"parent_id"` // For nested categories
	Icon      string `json:"icon"`      // CSS class or URL
	Color     string `json:"color"`     // Hex code
	SortOrder int    `json:"sort_order"`
	IsActive  bool   `json:"is_active"`
}

// Service represents a service offering
type Service struct {
	ID              string  `json:"id"`
	CategoryID      string  `json:"category_id"`
	Name            string  `json:"name"`
	BasePrice       float64 `json:"base_price"`
	DurationMinutes int     `json:"duration_minutes"` // Estimated time

	// [UPGRADE]
	WarrantyMonths int     `json:"warranty_months"`
	CommissionRate float64 `json:"commission_rate"` // Default commission for this service (%)
	RequiredSkill  string  `json:"required_skill"`  // Skill tag required to perform
	IsActive       bool    `json:"is_active"`
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
	TechnicianID string  `json:"technician_id"`
	BookingID    string  `json:"booking_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Accuracy     float64 `json:"accuracy"`
	Timestamp    int64   `json:"timestamp"`
	Speed        float64 `json:"speed,omitempty"`
	Heading      float64 `json:"heading,omitempty"`
}

// TechStatus represents current technician's location and status
type TechStatus struct {
	TechnicianID   string  `json:"technician_id"`
	TechnicianName string  `json:"technician_name"`
	CurrentBooking string  `json:"current_booking"`
	Status         string  `json:"status"` // idle, moving, arrived, working, completed
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	LastUpdate     int64   `json:"last_update"`
	Distance       float64 `json:"distance,omitempty"` // Distance to customer location in meters
}

// GeofenceEvent represents a geofencing alert
type GeofenceEvent struct {
	Type         string  `json:"type"` // arrived, departed, geofence_enter
	TechnicianID string  `json:"technician_id"`
	BookingID    string  `json:"booking_id"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
	Distance     float64 `json:"distance"`
	Timestamp    int64   `json:"timestamp"`
}
