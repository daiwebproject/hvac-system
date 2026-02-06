# Real-Time Location Tracking System - Implementation Guide

## Architecture Overview

Hệ thống Real-time Tracking gồm 4 komponen chính:

```
┌─────────────────────────────────────────────────────────────────┐
│                    REAL-TIME TRACKING ARCHITECTURE              │
├─────────────────────────────────────────────────────────────────┤
│                                                                   │
│  1. Client Layer (JavaScript)                                    │
│     ├─ LocationTracker.js (geolocation.watchPosition)           │
│     ├─ MapTracker.js (Leaflet map updates)                     │
│     └─ Throttling 10-15s interval                              │
│                                                                   │
│  2. API Layer (HTTP REST)                                       │
│     ├─ POST /api/tech/location/update (throttled)             │
│     ├─ POST /api/tech/tracking/start (begin tracking)         │
│     ├─ POST /api/tech/tracking/stop (end tracking)            │
│     ├─ GET /api/locations (admin - all techs)                 │
│     └─ GET /api/bookings/{id}/tech-location (customer)        │
│                                                                   │
│  3. Server Layer (Go/PocketBase)                               │
│     ├─ LocationCache (in-memory tech locations)               │
│     ├─ LocationHandler (REST API endpoints)                    │
│     ├─ GeofenceDetection (distance calculation)               │
│     ├─ SegmentedBroker (event distribution)                   │
│     └─ Database (only save final location & journey)          │
│                                                                   │
│  4. Streaming Layer (SSE - Server-Sent Events)                 │
│     ├─ Admin Dashboard (all techs stream)                      │
│     ├─ Customer Tracking (single tech stream)                  │
│     └─ Real-time notifications (arrival, status changes)       │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## API Endpoints Reference

### Tech Endpoints (Technician App)

#### 1. Update Location (Called frequently)
```
POST /api/tech/location/update
Content-Type: application/json

{
  "technician_id": "tech-123",
  "booking_id": "booking-456",
  "latitude": 21.0285,
  "longitude": 105.8542,
  "accuracy": 15.5,
  "speed": 5.2,
  "heading": 45.0
}

Response (200):
{
  "status": "success",
  "message": "Location updated",
  "tech_id": "tech-123",
  "booking_id": "booking-456",
  "distance": 250.5,
  "arrived": false
}
```

#### 2. Start Tracking
```
POST /api/tech/tracking/start
Content-Type: application/x-www-form-urlencoded

technician_id=tech-123&booking_id=booking-456

Response (200):
{
  "status": "Tracking started"
}
```

#### 3. Stop Tracking
```
POST /api/tech/tracking/stop
Content-Type: application/x-www-form-urlencoded

technician_id=tech-123&booking_id=booking-456

Response (200):
{
  "status": "Tracking stopped"
}
```

### Admin Endpoints

#### Get All Active Technician Locations
```
GET /api/admin/api/locations

Response (200):
{
  "count": 3,
  "techs": [
    {
      "technician_id": "tech-1",
      "technician_name": "Nguyễn Văn A",
      "current_booking": "booking-123",
      "status": "moving",
      "latitude": 21.0285,
      "longitude": 105.8542,
      "last_update": 1707242000,
      "distance": 250.5
    }
  ]
}
```

#### Stream Admin Location Updates (SSE)
```
GET /api/admin/api/locations/stream

(Opens SSE connection)
data: {
  "type": "location.updated",
  "timestamp": 1707242000,
  "data": {
    "technician_id": "tech-1",
    "technician_name": "Nguyễn Văn A",
    "booking_id": "booking-123",
    "latitude": 21.0285,
    "longitude": 105.8542,
    "distance": 250.5
  }
}

data: {
  "type": "geofence.arrived",
  "timestamp": 1707242010,
  "data": {
    "technician_id": "tech-1",
    "booking_id": "booking-123",
    "customer_name": "Nguyễn Thị B",
    "distance": 45.2,
    "message": "Thợ đã đến cổng nhà bạn"
  }
}
```

### Customer Endpoints

#### Get Technician Location (Current)
```
GET /api/bookings/booking-456/tech-location

Response (200):
{
  "technician_id": "tech-1",
  "technician_name": "Nguyễn Văn A",
  "current_booking": "booking-456",
  "status": "moving",
  "latitude": 21.0285,
  "longitude": 105.8542,
  "last_update": 1707242000,
  "distance": 250.5
}
```

#### Stream Technician Location (SSE)
```
GET /api/bookings/booking-456/location/stream

(Opens SSE connection)
data: {
  "type": "location.updated",
  "timestamp": 1707242000,
  "data": {
    "technician_id": "tech-1",
    "technician_name": "Nguyễn Văn A",
    "latitude": 21.0285,
    "longitude": 105.8542,
    "distance": 250.5
  }
}

data: {
  "type": "geofence.arrived",
  "timestamp": 1707242010,
  "data": {
    "message": "Thợ đã đến cổng nhà bạn"
  }
}
```

## Frontend Implementation

### For Technician App

#### 1. Initialize and Start Tracking
```javascript
// Import the tracking services
// <script src="/assets/js/services/location-tracking.js"></script>

// Initialize tracker
const tracker = new LocationTracker(
  'tech-id-from-session',
  'booking-id',
  {
    throttleInterval: 10000, // 10 seconds
    highAccuracyMode: true,
    onLocationUpdate: (data) => {
      console.log('Location updated:', data);
      if (data.success) {
        updateUIWithDistance(data.response.distance, data.response.arrived);
      }
    },
    onArrived: (data) => {
      showArrivalNotification(data);
      // Auto-update status or show next action
    },
    onError: (error) => {
      showErrorToast(error.message);
    },
    onStatusChange: (status) => {
      updateTrackingStatus(status);
    }
  }
);

// Request location permission first
if (!LocationTracker.isSupported()) {
  alert('Thiết bị này không hỗ trợ định vị GPS');
  exit();
}

const hasPermission = await LocationTracker.requestPermission();
if (!hasPermission) {
  alert('Vui lòng bật quyền truy cập vị trí trong cài đặt');
  exit();
}

// Start tracking when user clicks "Bắt đầu di chuyển"
document.getElementById('btn-start-moving').addEventListener('click', async () => {
  const success = await tracker.startTracking();
  if (success) {
    document.body.classList.add('tracking-active');
  }
});

// Stop tracking when job is completed
document.getElementById('btn-complete-job').addEventListener('click', async () => {
  const success = await tracker.stopTracking();
  if (success) {
    document.body.classList.remove('tracking-active');
  }
});
```

#### 2. Show Distance on Map
```javascript
// Import map tracking service
// <script src="/assets/js/services/map-tracking.js"></script>

// Initialize map tracker
const mapTracker = new MapTracker(
  leafletMapInstance,
  {
    interpolationDuration: 8000, // 8 seconds smooth animation
    onMarkerClick: (marker) => {
      console.log('Clicked:', marker);
    },
    onArrived: (data) => {
      showArrivedBanner(data.message);
    }
  }
);

// When tech location updates, update the map
tracker.onLocationUpdate = (locationData) => {
  if (locationData.success) {
    mapTracker.updateTechnicianLocation(
      'tech-id',
      'Technician Name',
      locationData.latitude,
      locationData.longitude,
      customerLat,  // From booking
      customerLng,  // From booking
      locationData.response.distance
    );
  }
};
```

### For Admin Dashboard

#### Stream All Technician Locations
```javascript
// Using EventSource (native SSE API)
const source = new EventSource('/api/admin/api/locations/stream');

// Initialize map tracker
const mapTracker = new MapTracker(leafletMapInstance);

source.addEventListener('location.updated', (event) => {
  const data = JSON.parse(event.data);
  mapTracker.updateTechnicianLocation(
    data.data.technician_id,
    data.data.technician_name,
    data.data.latitude,
    data.data.longitude,
    null,  // No customer location needed for admin
    null,
    data.data.distance
  );
  
  updateTechListUI(data.data);
});

source.addEventListener('geofence.arrived', (event) => {
  const data = JSON.parse(event.data);
  mapTracker.showArrivalNotification(
    data.data.technician_id,
    data.data.technician_name,
    data.data.message
  );
  updateBookingStatus(data.data.booking_id, 'arrived');
});

source.addEventListener('error', (event) => {
  if (event.eventPhase === EventSource.CLOSED) {
    console.log('SSE connection closed');
  }
});
```

### For Customer Tracking Page

#### Stream Single Technician Location
```javascript
const bookingId = document.getElementById('booking-id').value;
const customerLat = parseFloat(document.getElementById('customer-lat').value);
const customerLng = parseFloat(document.getElementById('customer-lng').value);

// Fetch initial location
const initialLocation = await fetch(`/api/bookings/${bookingId}/tech-location`).then(r => r.json());

// Initialize map with customer location
const mapTracker = new MapTracker(leafletMapInstance);
mapTracker.updateCustomerMarker(bookingId, customerLat, customerLng);

// Update with initial tech location
if (initialLocation) {
  mapTracker.updateTechnicianLocation(
    initialLocation.technician_id,
    initialLocation.technician_name,
    initialLocation.latitude,
    initialLocation.longitude,
    customerLat,
    customerLng,
    initialLocation.distance
  );
  
  document.getElementById('distance-display').textContent = 
    mapTracker.formatDistance(initialLocation.distance);
}

// Stream real-time updates
const source = new EventSource(`/api/bookings/${bookingId}/location/stream`);

source.addEventListener('location.updated', (event) => {
  const data = JSON.parse(event.data);
  mapTracker.updateTechnicianLocation(
    data.data.technician_id,
    data.data.technician_name,
    data.data.latitude,
    data.data.longitude,
    customerLat,
    customerLng,
    data.data.distance
  );
  
  document.getElementById('distance-display').textContent = 
    mapTracker.formatDistance(data.data.distance);
  
  document.getElementById('last-update').textContent = 
    new Date(data.data.timestamp * 1000).toLocaleTimeString('vi-VN');
});

source.addEventListener('geofence.arrived', (event) => {
  const data = JSON.parse(event.data);
  mapTracker.showArrivalNotification(
    data.data.technician_id,
    data.data.technician_name,
    data.data.message
  );
  
  // Show "Thợ đã đến" banner
  document.getElementById('arrival-banner').style.display = 'block';
});

source.onerror = () => {
  if (source.readyState === EventSource.CLOSED) {
    console.log('Connection closed by server');
  }
};
```

## Backend Data Flow

### Location Update Flow
```
1. Client sends location via POST /api/tech/location/update
   ↓
2. LocationHandler.UpdateLocation() receives and validates
   ↓
3. LocationCache.UpdateTechLocation() stores in memory with throttling
   ↓
4. If throttle passed (10+ seconds):
   ├─ Calculate distance to customer
   ├─ Check geofence (< 100m)
   └─ Broadcast via SegmentedBroker
   ↓
5. Broker.Publish() sends to:
   ├─ Admin channel (all locations)
   ├─ Customer channel {booking_id} (their tech)
   └─ Tech channel {tech_id} (personal)
   ↓
6. SSE streams pick up events and send to connected clients
```

### Geofence Detection Flow
```
1. Location update received
   ↓
2. LocationCache.CheckGeofence() calculates distance
   ↓
3. If distance < 100m && status == "moving":
   ├─ Update booking status to "arrived" in DB
   └─ Broker.Publish() geofence.arrived event
   ↓
4. Admin + Customer SSE streams receive event
   ↓
5. Frontend shows notification and updates UI
```

## Important Notes & Best Practices

### 1. Throttling Strategy
- **10 seconds minimum** between database/broadcast updates
- **watchPosition** fires only when position changes (efficient)
- **First update sent immediately** to get initial location fast

### 2. In-Memory Cache vs Database
- **Cache**: Real-time tech locations (memory efficient)
- **Database**: Only final location when job completes
- **No continuous writes** to avoid slowing down DB

### 3. Geofencing Parameters
```go
// Default 100 meters for "arrived" detection
geofenceRadius := 100.0

// Customize if needed:
// - Busy urban areas: 50-75m (narrow streets)
// - Suburban areas: 100-150m
// - Rural: 200m+
```

### 4. Security Considerations

Add authentication to all endpoints:
```javascript
// Include token in requests
const headers = {
  'Content-Type': 'application/json',
  'Authorization': `Bearer ${sessionToken}`
};

fetch('/api/tech/location/update', {
  method: 'POST',
  headers: headers,
  body: JSON.stringify(location)
});
```

### 5. Battery Optimization
- `watchPosition` is more battery-efficient than `setInterval`
- High accuracy mode uses more power (only enable on moving)
- Stop tracking when job completes to save battery

### 6. Error Handling & Fallback
```javascript
// If geolocation fails
if (error.code === error.PERMISSION_DENIED) {
  // User denied permission
  showAlert('Vui lòng bật quyền truy cập vị trí');
  tracker.stopTracking();
}

if (error.code === error.POSITION_UNAVAILABLE) {
  // GPS signal lost
  console.warn('Mất tín hiệu GPS, sẽ thử lại...');
  // Continue trying as watchPosition retries
}

// Server unreachable - queue updates locally
if (locationData.status === 'throttled') {
  // Update still being queued, not sent yet
  console.log('Cập nhật sẽ được gửi trong', 
    10 - ((Date.now() - lastSent) / 1000), 'giây');
}
```

## Testing Checklist

- [ ] Tech can start/stop tracking
- [ ] Locations update in real-time on admin map
- [ ] Customer can see tech approaching
- [ ] Arrival notification triggers at 100m
- [ ] Booking status auto-updates to "arrived"
- [ ] Distance displayed correctly
- [ ] SSE streams handle disconnects gracefully
- [ ] No database bloat from location saves
- [ ] Works offline with queuing (if implemented)
- [ ] Battery usage reasonable

## Performance Tuning

### Optimize for 100+ Concurrent Technicians
```go
// Increase SSE broadcast buffer
clientChan := make(chan Event, 50) // Increase from 10

// Cache tuning
locationCache := services.NewLocationCache()
// Memory: ~1KB per technician = 100KB for 100 techs

// Database indexes
// Create index on: bookings(technician_id, job_status)
// Create index on: bookings(created)
```

## Troubleshooting

| Issue | Solution |
|-------|----------|
| SSE not connecting | Check CORS headers, ensure `/api/locations/stream` is accessible |
| Location updates lagging | Increase throttle interval (currently 10s), check network |
| High memory usage | Implement location history cleanup (remove old paths) |
| Battery drain | Disable high accuracy mode when not moving |
| Marker not moving smoothly | Check interpolation duration (8s default) |

## Files Created/Modified

### New Files
- `pkg/services/location_cache.go` - In-memory location storage
- `internal/handler/location_handler.go` - REST API endpoints
- `internal/handler/location_sse_handler.go` - SSE streaming
- `assets/js/services/location-tracking.js` - Client geolocation
- `assets/js/services/map-tracking.js` - Leaflet map updates

### Modified Files
- `main.go` - Initialize location components
- `pkg/app/router.go` - Register all routes
- `internal/core/models.go` - LocationUpdate, TechStatus models
- `internal/core/ports.go` - BookingRepository.UpdateStatus/Location
- `internal/adapter/repository/booking_repo.go` - Implement new methods

---

**More questions?** Check the API endpoint structure and ensure all routes are correctly registered in `router.go`.
