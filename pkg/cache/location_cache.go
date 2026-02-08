package cache

import (
	"hvac-system/internal/core"
	"math"
	"sync"
	"time"
)

// LocationCache stores real-time technician location data in memory
// This avoids writing every location update to the database
type LocationCache struct {
	// Map[tech_id] -> TechStatus
	techLocations map[string]*core.TechStatus
	mutex         sync.RWMutex

	// Track last reported location timestamp per tech to throttle updates
	lastReportTime map[string]int64
	reportMutex    sync.RWMutex
}

// NewLocationCache creates a new location cache instance
func NewLocationCache() *LocationCache {
	return &LocationCache{
		techLocations:  make(map[string]*core.TechStatus),
		lastReportTime: make(map[string]int64),
	}
}

// UpdateTechLocation updates/creates a technician's current location
// Returns true if this is a new update (past throttle period)
func (lc *LocationCache) UpdateTechLocation(
	techID string,
	bookingID string,
	lat float64,
	lng float64,
	accuracy float64,
	speed float64,
	heading float64,
) (bool, *core.TechStatus) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	now := time.Now().UnixMilli()

	// Check throttle - only update if 2 seconds have passed
	lc.reportMutex.RLock()
	lastTime := lc.lastReportTime[techID]
	lc.reportMutex.RUnlock()

	isNewUpdate := (now - lastTime) >= 2000 // 2 seconds

	status := lc.techLocations[techID]
	if status == nil {
		status = &core.TechStatus{
			TechnicianID: techID,
			Status:       "idle",
		}
	}

	// Update location
	status.Latitude = lat
	status.Longitude = lng
	status.LastUpdate = now
	status.CurrentBooking = bookingID

	lc.techLocations[techID] = status

	// Track last report time
	if isNewUpdate {
		lc.reportMutex.Lock()
		lc.lastReportTime[techID] = now
		lc.reportMutex.Unlock()
	}

	return isNewUpdate, status
}

// GetTechLocation retrieves current location of a technician
func (lc *LocationCache) GetTechLocation(techID string) *core.TechStatus {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	return lc.techLocations[techID]
}

// GetAllActiveTechs returns all technicians with active bookings
func (lc *LocationCache) GetAllActiveTechs() []*core.TechStatus {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	var result []*core.TechStatus
	for _, status := range lc.techLocations {
		if status.CurrentBooking != "" && status.Status != "idle" {
			result = append(result, status)
		}
	}
	return result
}

// GetTechsByBooking returns technician location for a specific booking
func (lc *LocationCache) GetTechsByBooking(bookingID string) []*core.TechStatus {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	var result []*core.TechStatus
	for _, status := range lc.techLocations {
		if status.CurrentBooking == bookingID {
			result = append(result, status)
		}
	}
	return result
}

// UpdateTechStatus updates technician's current status
func (lc *LocationCache) UpdateTechStatus(techID string, status string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if tech, exists := lc.techLocations[techID]; exists {
		tech.Status = status
	}
}

// SetTechnicianInfo sets technician name (called once when tech logs in)
func (lc *LocationCache) SetTechnicianInfo(techID string, name string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if status, exists := lc.techLocations[techID]; exists {
		status.TechnicianName = name
	}
}

// ClearTechLocation removes a technician from cache
func (lc *LocationCache) ClearTechLocation(techID string) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	delete(lc.techLocations, techID)

	lc.reportMutex.Lock()
	delete(lc.lastReportTime, techID)
	lc.reportMutex.Unlock()
}

// ============ GEOFENCING UTILITIES ============

// CalculateDistance calculates distance between two coordinates in meters using Haversine formula
func CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusM = 6371000.0 // Earth radius in meters

	dLat := degreesToRadians(lat2 - lat1)
	dLng := degreesToRadians(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degreesToRadians(lat1))*math.Cos(degreesToRadians(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusM * c
}

// degreesToRadians converts degrees to radians
func degreesToRadians(degrees float64) float64 {
	return degrees * math.Pi / 180.0
}

// CheckGeofence checks if technician has arrived at customer location
// Returns (arrived, distance)
func (lc *LocationCache) CheckGeofence(
	techID string,
	customerLat float64,
	customerLng float64,
	geofenceRadius float64, // in meters, default 100m for "arrived"
) (bool, float64) {
	lc.mutex.RLock()
	defer lc.mutex.RUnlock()

	status, exists := lc.techLocations[techID]
	if !exists {
		return false, 0
	}

	distance := CalculateDistance(status.Latitude, status.Longitude, customerLat, customerLng)
	arrived := distance < geofenceRadius

	return arrived, distance
}

// UpdateDistance updates cached distance for a technician
func (lc *LocationCache) UpdateDistance(techID string, distance float64) {
	lc.mutex.Lock()
	defer lc.mutex.Unlock()

	if status, exists := lc.techLocations[techID]; exists {
		status.Distance = distance
	}
}
