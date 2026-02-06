package handler

import (
	"encoding/json"
	"hvac-system/internal/core"
	"hvac-system/pkg/broker"
	"hvac-system/pkg/services"
	"log"
	"time"

	pbCore "github.com/pocketbase/pocketbase/core"
)

type LocationHandler struct {
	locationCache  *services.LocationCache
	bookingRepo    core.BookingRepository
	techRepo       core.TechnicianRepository
	broker         *broker.SegmentedBroker
	geofenceRadius float64 // Default 100 meters
}

func NewLocationHandler(
	cache *services.LocationCache,
	bookingRepo core.BookingRepository,
	techRepo core.TechnicianRepository,
	broker *broker.SegmentedBroker,
) *LocationHandler {
	return &LocationHandler{
		locationCache:  cache,
		bookingRepo:    bookingRepo,
		techRepo:       techRepo,
		broker:         broker,
		geofenceRadius: 100.0, // 100 meters default
	}
}

// UpdateLocation handles POST /api/location
// Request body: {
//   "technician_id": "tech123",
//   "booking_id": "booking456",
//   "latitude": 21.0285,
//   "longitude": 105.8542,
//   "accuracy": 15.5,
//   "speed": 5.2,
//   "heading": 45.0
// }
func (h *LocationHandler) UpdateLocation(e *pbCore.RequestEvent) error {
	var req struct {
		TechnicianID string  `json:"technician_id"`
		BookingID    string  `json:"booking_id"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
		Accuracy     float64 `json:"accuracy"`
		Speed        float64 `json:"speed"`
		Heading      float64 `json:"heading"`
	}

	if err := json.NewDecoder(e.Request.Body).Decode(&req); err != nil {
		return e.JSON(400, map[string]string{"error": "Invalid request body"})
	}

	// Validate required fields
	if req.TechnicianID == "" || req.BookingID == "" {
		return e.JSON(400, map[string]string{"error": "Missing technician_id or booking_id"})
	}

	if req.Latitude == 0 || req.Longitude == 0 {
		return e.JSON(400, map[string]string{"error": "Invalid coordinates"})
	}

	// Update cache with throttling
	isNewUpdate, techStatus := h.locationCache.UpdateTechLocation(
		req.TechnicianID,
		req.BookingID,
		req.Latitude,
		req.Longitude,
		req.Accuracy,
		req.Speed,
		req.Heading,
	)

	// Only broadcast if this is a new update (past throttle period)
	if !isNewUpdate {
		return e.JSON(200, map[string]interface{}{
			"status":     "throttled",
			"message":    "Location received but throttled (update too frequent)",
			"tech_id":    req.TechnicianID,
			"booking_id": req.BookingID,
		})
	}

	// Fetch booking info
	booking, err := h.bookingRepo.GetByID(req.BookingID)
	if err != nil {
		log.Printf("⚠️ Booking not found: %s", req.BookingID)
		// Still broadcast location, but without booking details
	}

	// Calculate distance to customer
	var distance float64
	var arrived bool
	if booking != nil {
		distance = services.CalculateDistance(
			req.Latitude,
			req.Longitude,
			booking.Lat,
			booking.Long,
		)
		h.locationCache.UpdateDistance(req.TechnicianID, distance)

		// Check geofence
		arrived, _ = h.locationCache.CheckGeofence(
			req.TechnicianID,
			booking.Lat,
			booking.Long,
			h.geofenceRadius,
		)
	}

	// ============ BROADCAST LOCATION UPDATE ============

	locationEvent := broker.Event{
		Type:      "location.updated",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"technician_id":   req.TechnicianID,
			"technician_name": techStatus.TechnicianName,
			"booking_id":      req.BookingID,
			"latitude":        req.Latitude,
			"longitude":       req.Longitude,
			"accuracy":        req.Accuracy,
			"speed":           req.Speed,
			"heading":         req.Heading,
			"distance":        distance,
			"timestamp":       time.Now().UnixMilli(),
		},
	}

	// 1. Broadcast to Admin (all locations)
	if h.broker != nil {
		h.broker.Publish(broker.ChannelAdmin, "", locationEvent)

		// 2. Broadcast to Customer (only their booking's tech)
		if booking != nil {
			h.broker.Publish(broker.ChannelCustomer, req.BookingID, locationEvent)
		}

		// 3. Broadcast to Technician SSE if exists
		h.broker.Publish(broker.ChannelTech, req.TechnicianID, locationEvent)
	}

	// ============ GEOFENCE CHECK: ARRIVED DETECTION ============
	if arrived && booking != nil && booking.JobStatus == "moving" {
		log.Printf("✅ [GEOFENCE] Tech %s has ARRIVED at booking %s (distance: %.2f m)", 
			req.TechnicianID, req.BookingID, distance)

		// Update booking status to "arrived"
		if err := h.bookingRepo.UpdateStatus(req.BookingID, "arrived"); err != nil {
			log.Printf("❌ Failed to update booking status: %v", err)
		}

		// Publish geofence event
		geofenceEvent := broker.Event{
			Type:      "geofence.arrived",
			Timestamp: time.Now().Unix(),
			Data: map[string]interface{}{
				"technician_id":   req.TechnicianID,
				"technician_name": techStatus.TechnicianName,
				"booking_id":      req.BookingID,
				"customer_name":   booking.CustomerName,
				"distance":        distance,
				"message":         "Thợ đã đến cổng nhà bạn",
			},
		}

		// Broadcast to customer
		if h.broker != nil {
			h.broker.Publish(broker.ChannelCustomer, req.BookingID, geofenceEvent)
			// Also to admin
			h.broker.Publish(broker.ChannelAdmin, "", geofenceEvent)
		}
	}

	return e.JSON(200, map[string]interface{}{
		"status":     "success",
		"message":    "Location updated",
		"tech_id":    req.TechnicianID,
		"booking_id": req.BookingID,
		"distance":   distance,
		"arrived":    arrived,
	})
}

// GetTechLocation handles GET /api/location/:tech_id
func (h *LocationHandler) GetTechLocation(e *pbCore.RequestEvent) error {
	techID := e.Request.PathValue("id")
	if techID == "" {
		return e.JSON(400, map[string]string{"error": "Missing technician ID"})
	}

	status := h.locationCache.GetTechLocation(techID)
	if status == nil {
		return e.JSON(404, map[string]string{"error": "Technician location not found"})
	}

	return e.JSON(200, status)
}

// GetAllTechLocations handles GET /api/locations (for admin dashboard)
// Returns all active technicians with their current locations
func (h *LocationHandler) GetAllTechLocations(e *pbCore.RequestEvent) error {
	techs := h.locationCache.GetAllActiveTechs()
	return e.JSON(200, map[string]interface{}{
		"count": len(techs),
		"techs": techs,
	})
}

// GetBookingTechLocation handles GET /api/bookings/:booking_id/tech-location
// Returns technician location for a specific booking (for customer view)
func (h *LocationHandler) GetBookingTechLocation(e *pbCore.RequestEvent) error {
	bookingID := e.Request.PathValue("id")
	if bookingID == "" {
		return e.JSON(400, map[string]string{"error": "Missing booking ID"})
	}

	techs := h.locationCache.GetTechsByBooking(bookingID)
	if len(techs) == 0 {
		return e.JSON(404, map[string]string{"error": "No technician assigned to this booking"})
	}

	// Return the first (and should be only) technician for this booking
	return e.JSON(200, techs[0])
}

// StartTracking handles POST /api/tracking/start
// Called when tech clicks "Bắt đầu di chuyển" button
func (h *LocationHandler) StartTracking(e *pbCore.RequestEvent) error {
	techID := e.Request.FormValue("technician_id")
	bookingID := e.Request.FormValue("booking_id")

	if techID == "" || bookingID == "" {
		return e.JSON(400, map[string]string{"error": "Missing required fields"})
	}

	h.locationCache.UpdateTechStatus(techID, "moving")

	// Update booking status
	if err := h.bookingRepo.UpdateStatus(bookingID, "moving"); err != nil {
		return e.JSON(500, map[string]string{"error": err.Error()})
	}

	// Notify
	startEvent := broker.Event{
		Type:      "tracking.started",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"technician_id": techID,
			"booking_id":    bookingID,
			"message":       "Thợ đang trên đường đến nhà bạn",
		},
	}

	if h.broker != nil {
		h.broker.Publish(broker.ChannelCustomer, bookingID, startEvent)
		h.broker.Publish(broker.ChannelAdmin, "", startEvent)
	}

	return e.JSON(200, map[string]string{"status": "Tracking started"})
}

// StopTracking handles POST /api/tracking/stop
// Called when job is completed
func (h *LocationHandler) StopTracking(e *pbCore.RequestEvent) error {
	techID := e.Request.FormValue("technician_id")
	bookingID := e.Request.FormValue("booking_id")

	if techID == "" || bookingID == "" {
		return e.JSON(400, map[string]string{"error": "Missing required fields"})
	}

	// Get final location for record
	techStatus := h.locationCache.GetTechLocation(techID)
	
	// Save final location to booking (optional)
	if _, err := h.bookingRepo.GetByID(bookingID); err == nil && techStatus != nil {
		// Update final coordinates
		if err := h.bookingRepo.UpdateLocation(bookingID, techStatus.Latitude, techStatus.Longitude); err != nil {
			log.Printf("⚠️ Failed to save final location: %v", err)
		}
	}

	// Clear from active cache
	h.locationCache.ClearTechLocation(techID)

	// Notify
	stopEvent := broker.Event{
		Type:      "tracking.stopped",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"technician_id": techID,
			"booking_id":    bookingID,
			"status":        "completed",
		},
	}

	if h.broker != nil {
		h.broker.Publish(broker.ChannelCustomer, bookingID, stopEvent)
		h.broker.Publish(broker.ChannelAdmin, "", stopEvent)
	}

	return e.JSON(200, map[string]string{"status": "Tracking stopped"})
}

// HealthCheck handles GET /api/health/location
// For monitoring location service health
func (h *LocationHandler) HealthCheck(e *pbCore.RequestEvent) error {
	activeTechs := h.locationCache.GetAllActiveTechs()
	
	return e.JSON(200, map[string]interface{}{
		"service":      "location-tracker",
		"status":       "healthy",
		"active_techs": len(activeTechs),
		"timestamp":    time.Now().Unix(),
	})
}
