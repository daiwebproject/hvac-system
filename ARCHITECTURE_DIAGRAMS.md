# Real-Time Location Tracking - Architecture Diagrams

## System Architecture Diagram

```
┌────────────────────────────────────────────────────────────────────────┐
│                         HVAC BOOKING APPLICATION                       │
├────────────────────────────────────────────────────────────────────────┤
│                                                                          │
│  ┌─────────────────────┐  ┌──────────────────────┐  ┌──────────────┐   │
│  │   TECH APP          │  │   ADMIN WEB          │  │  CUSTOMER    │   │
│  │   (Mobile/Web)      │  │   (Dashboard)        │  │  (Website)   │   │
│  ├─────────────────────┤  ├──────────────────────┤  ├──────────────┤   │
│  │ • LocationTracker   │  │ • MapTracker         │  │ • Booking    │   │
│  │ • watchPosition()   │  │ • SSE Stream         │  │   Tracker    │   │
│  │ • Start/Stop Track  │  │ • All Tech Markers   │  │ • Live Map   │   │
│  │ • Show Distance     │  │ • Arrival Alert      │  │ • ETA        │   │
│  └─────────────────────┘  └──────────────────────┘  └──────────────┘   │
│          │                        │                        │             │
│          │                        │                        │             │
│  ┌───────▼─────────────────────────▼────────────────────────▼────────┐  │
│  │                    HTTP API LAYER                                 │  │
│  ├────────────────────────────────────────────────────────────────┤  │
│  │                                                                 │  │
│  │ Public:    /api/health/location                              │  │
│  │            /api/bookings/{id}/tech-location                 │  │
│  │            /api/tech/{id}/events/stream (SSE)               │  │
│  │                                                                 │  │
│  │ Tech:      POST /api/tech/location/update                   │  │
│  │            POST /api/tech/tracking/start                    │  │
│  │            POST /api/tech/tracking/stop                     │  │
│  │            GET  /api/tech/location                          │  │
│  │                                                                 │  │
│  │ Admin:     GET  /api/admin/locations                        │  │
│  │            GET  /api/admin/locations/stream (SSE)           │  │
│  │                                                                 │  │
│  └───────┬────────────────────────────────────────────────────────┘  │
│          │                                                              │
│  ┌───────▼────────────────────────────────────────────────────┐  │
│  │            LOCATION HANDLER (Go REST)                      │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │                                                               │  │
│  │ • Validate location data                                   │  │
│  │ • Check throttle (10s minimum)                            │  │
│  │ • Delegate to cache                                       │  │
│  │ • Calculate distance                                      │  │
│  │ • Check geofence                                          │  │
│  │ • Publish events                                          │  │
│  │                                                               │  │
│  └───────┬────────────────────────────────────────────────────┘  │
│          │                                                          │
│  ┌───────▼────────────────────────────────────────────────────┐  │
│  │         LOCATION CACHE (In-Memory)                         │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │                                                               │  │
│  │ Map[tech_id] → TechStatus {                                │  │
│  │   lat, lng, status, distance, last_update                │  │
│  │ }                                                             │  │
│  │                                                               │  │
│  │ Per-tech throttle: last_report_time                        │  │
│  │                                                               │  │
│  │ Geofence: CalculateDistance() & CheckGeofence()           │  │
│  │                                                               │  │
│  └───────┬────────────────────────────────────────────────────┘  │
│          │                                                          │
│  ┌───────▼────────────────────────────────────────────────────┐  │
│  │      SEGMENTED BROKER (Event Distribution)                 │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │                                                               │  │
│  │ Admin Channel     ← All tech locations                     │  │
│  │ Customer Channel  ← Specific booking's tech               │  │
│  │ Tech Channel      ← Personal job events                    │  │
│  │                                                               │  │
│  │ Events:                                                       │  │
│  │  • location.updated      (every 10s)                       │  │
│  │  • geofence.arrived      (when < 100m)                    │  │
│  │  • tracking.started      (user action)                     │  │
│  │  • tracking.stopped      (job complete)                    │  │
│  │                                                               │  │
│  └───────┬────────────────────────────────────────────────────┘  │
│          │                                                          │
│  ┌───────▼────────────────────────────────────────────────────┐  │
│  │       SSE HANDLER (Server-Sent Events Streaming)           │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │                                                               │  │
│  │ Admin Stream:                                               │  │
│  │  ├─ Client 1 (Admin User #1)                               │  │
│  │  ├─ Client 2 (Admin User #2)                               │  │
│  │  └─ ...                                                      │  │
│  │                                                               │  │
│  │ Customer Streams (per booking):                             │  │
│  │  ├─ Booking-123 ─ Client 1                                 │  │
│  │  ├─ Booking-456 ─ Client 1                                 │  │
│  │  └─ ...                                                      │  │
│  │                                                               │  │
│  │ Tech Streams (per technician):                              │  │
│  │  ├─ Tech-001 ─ Client 1                                    │  │
│  │  ├─ Tech-002 ─ Client 1                                    │  │
│  │  └─ ...                                                      │  │
│  │                                                               │  │
│  └───────┬────────────────────────────────────────────────────┘  │
│          │                                                          │
│  ┌───────▼────────────────────────────────────────────────────┐  │
│  │        DATABASE (PocketBase)                               │  │
│  ├───────────────────────────────────────────────────────────┤  │
│  │                                                               │  │
│  │ bookings table:                                             │  │
│  │  ├─ job_status    (pending → moving → arrived → ...)       │  │
│  │  ├─ lat, lng      (final location when complete)           │  │
│  │  ├─ arrived_at    (timestamp of arrival)                   │  │
│  │  ├─ completed_at  (timestamp of completion)                │  │
│  │  └─ ...                                                      │  │
│  │                                                               │  │
│  │ ⚠️  Location updates NOT written to DB (in-memory only)    │  │
│  │                                                               │  │
│  └───────────────────────────────────────────────────────────┘  │
│                                                                      │
└──────────────────────────────────────────────────────────────────────┘
```

---

## Location Update Message Flow

```
TIME    ACTIVITY                          CACHE STATE              DB STATE

0:00    Tech starts job at booking-123
        └─→ POST /api/tech/tracking/start
            └─→ Handler updates cache
                ├─ Status: "moving"
                ├─ Booking: "booking-123"
                └─ First location pending

0:05    GPS gets first lock
        └─→ LocationTracker.handlePositionUpdate()
            └─→ First update sends immediately
                POST /api/tech/location/update
                ├─ lat: 21.0285, lng: 105.8542
                └─ Broadcast to broker

        Cache:
        tech-001: {
          status: "moving",
          lat: 21.0285,
          lng: 105.8542,
          distance: 450m,
          last_update: NOW
        }

0:08    GPS updates position again              ← Ignored (< 10s)

0:12    GPS updates, 10+ seconds passed        
        └─→ POST /api/tech/location/update
            ├─ lat: 21.0287, lng: 105.8545
            ├─ throttle: PASSED
            └─ Broadcast
                ├─→ Admin (all locations)
                ├─→ Customer {booking-123}
                └─→ Tech {tech-001}

        Cache:
        tech-001: {
          status: "moving",
          lat: 21.0287,
          lng: 105.8545,
          distance: 425m,  ← Updated
          last_update: NOW
        }

0:22    GPS updates
        └─→ POST /api/tech/location/update
            ├─ lat: 21.0290, lng: 105.8550
            └─ GEOFENCE CHECK
                distance < 100m? YES!
                status == "moving"? YES!
                
                ├─→ Handler: UpdateStatus("arrived")
                │   └─→ DB UPDATE
                │       bookings.job_status = "arrived"
                │       bookings.arrived_at = NOW
                │
                └─→ Broker: Publish geofence.arrived
                    ├─→ Admin SSE (notification)
                    └─→ Customer SSE (notification + banner)

        Cache:
        tech-001: {
          status: "arrived",      ← Updated
          lat: 21.0290,
          lng: 105.8550,
          distance: 45m,          ← Updated
          last_update: NOW
        }

        Database:                   ← FIRST DB WRITE
        UPDATE bookings
        SET job_status = 'arrived',
            arrived_at = NOW()
        WHERE id = 'booking-123'

... technician works on job ...

1:15    Tech completes job
        └─→ POST /api/tech/tracking/stop
            ├─ Delete from cache
            ├─ DB UPDATE: job_status = 'completed'
            ├─ DB SAVE: final lat/lng
            └─ Broker: Publish tracking.stopped
                ├─→ Admin SSE
                └─→ Customer SSE

        Cache:
        tech-001: REMOVED          ← Cache cleared

        Database:                   ← FINAL DB WRITE
        UPDATE bookings
        SET job_status = 'completed',
            completed_at = NOW(),
            lat = 21.0290,
            lng = 105.8550
        WHERE id = 'booking-123'
```

---

## Event Broadcasting Flow

```
┌─────────────────────┐
│ Location Received   │
└──────────┬──────────┘
           │
           ▼
┌──────────────────────────────────────┐
│ LocationHandler.UpdateLocation()      │
│                                        │
│ 1. Validate input                    │
│ 2. Check throttle (10s)              │
│ 3. Update cache                      │
│ 4. Calculate distance                │
│ 5. Check geofence                    │
└──────────────┬───────────────────────┘
               │
               ▼
      ┌────────────────────┐
      │ Throttle Passed?   │
      └────┬───────────┬───┘
           │ YES       │ NO
           │           │
    ┌──────▼──┐    ┌───▼────────────┐
    │Broadcast│    │Return "throttled"
    └──────┬──┘    └────────────────┘
           │
           ▼
    ┌─────────────────────────────┐
    │ SegmentedBroker.Publish()    │
    └─────────────────────────────┘
            │
     ┌──────┼──────┬──────────┐
     │      │      │          │
     ▼      ▼      ▼          ▼
  ADMIN  CUSTOMER TECH      (if arrived)
  CHANNEL CHANNEL  CHANNEL   GEOFENCE
                            EVENT
     │      │      │          │
     └──────┼──────┴──────────┘
            │
            ▼
    ┌─────────────────────────────┐
    │ SSE Handler Streams          │
    │ event: location.updated      │
    │ data: { ... }               │
    └─────────────────────────────┘
            │
     ┌──────┼──────┬──────────┐
     │      │      │          │
     ▼      ▼      ▼          ▼
  Admin   Customer Tech      Geofence
  Users   Users    App       Listeners
     │      │      │          │
     │      │      │          │
     ▼      ▼      ▼          ▼
  Map    Map    Log       Update
  Update Update Update    Status
```

---

## Client-Side Location Tracking Flow

```
┌────────────────────────────────┐
│ Tech Opens Job Detail Page     │
└───────────┬────────────────────┘
            │
            ▼
┌────────────────────────────────┐
│ Click "Bắt đầu di chuyển"      │
└───────────┬────────────────────┘
            │
            ▼
┌────────────────────────────────┐
│ LocationTracker.startTracking()│
│                                │
│ 1. POST /api/tracking/start    │
│ 2. watchPosition() activated   │
│ 3. lastSentTime = 0            │
└───────────┬────────────────────┘
            │
            ▼
    ┌───────────────────┐
    │GPS Enabled?       │
    └─┬─────────────┬───┘
      │ YES         │ NO
      │             │
      ▼        ┌────▼─────────────┐
  Position    │ showError()        │
  Available   │ stopTracking()     │
      │       └──────────────────┘
      │
      ▼
  watchPosition()
      │
      ├─→ handler: hanldePositionUpdate()
      │       │
      │       ▼
      │   Now - lastSent >= 10s?
      │       │
      │   ┌───┴────────┬─────────────┐
      │   │ YES        │ NO          │
      │   │            │             │
      │   │      updateUI("throttled")
      │   │
      │   ▼
      │   sendLocationToServer()
      │       │
      │       ▼
      │   POST /api/tech/location/update
      │       │
      │       ├─→ Success
      │       │   ├─ lastSentTime = now
      │       │   ├─ sentCount++
      │       │   └─ onLocationUpdate(data)
      │       │       ├─ updateUI(distance, speed)
      │       │       └─ mapTracker.update()
      │       │
      │       └─→ Error
      │           ├─ errorCount++
      │           └─ onError(error)
      │
      └─→ Continue watching (loop)

┌───────────────┐
│ Job Complete  │
└───────┬───────┘
        │
        ▼
┌─────────────────────────────────┐
│ Click "Hoàn thành công việc"     │
└───────┬───────────────────────────┘
        │
        ▼
┌─────────────────────────────────┐
│ LocationTracker.stopTracking()   │
│                                  │
│ 1. POST /api/tracking/stop      │
│ 2. clearWatch(watchId)          │
│ 3. isTracking = false           │
└───────┬───────────────────────────┘
        │
        ▼
┌─────────────────────────────────┐
│ Submit Job Completion           │
│ (existing flow)                 │
└─────────────────────────────────┘
```

---

## Geofence Detection Algorithm

```
Tech Location: (lat1, lng1)
Customer Location: (lat2, lng2)
Geofence Radius: 100 meters

Step 1: Calculate Haversine Distance
───────────────────────────────────────
dLat = (lat2 - lat1) * π / 180
dLng = (lng2 - lng1) * π / 180

a = sin²(dLat/2) + cos(lat1*π/180) * cos(lat2*π/180) * sin²(dLng/2)
c = 2 * atan2(√a, √(1-a))

distance = 6371000 * c  (6371000 = Earth radius in meters)

Step 2: Check Geofence
──────────────────────
if (distance < 100m) AND (status == "moving") then
    ✅ ARRIVED
    └─→ Update booking.job_status = "arrived"
    └─→ Publish geofence.arrived event
else
    ⏳ STILL MOVING or NOT MOVING
    └─→ Continue monitoring
```

---

## Database Schema Updates

```
bookings table:
┌─────────────────────────────────┐
│ EXISTING FIELDS                 │
├─────────────────────────────────┤
│ id (primary)                    │
│ service_id                      │
│ customer_name                   │
│ customer_phone                  │
│ address                         │
│ issue_description               │
│ technician_id (foreign key)    │
│ job_status                      │
│ booking_time                    │
│ created                         │
│ updated                         │
└─────────────────────────────────┘

┌─────────────────────────────────┐
│ NEW/UPDATED FIELDS              │
├─────────────────────────────────┤
│ lat (float)         ← Final loc │
│ long (float)        ← Final loc │
│ moved_start_at      ← Timestamp │
│ arrived_at          ← Timestamp │
│ working_start_at    ← Timestamp │
│ completed_at        ← Timestamp │
└─────────────────────────────────┘

Indices to add:
──────────────
CREATE INDEX idx_booking_technician 
ON bookings(technician_id, job_status);

CREATE INDEX idx_booking_created 
ON bookings(created DESC);
```

---

## Memory Usage Estimation

```
Location Cache (100 concurrent technicians)
──────────────────────────────────────────

Per technician entry:
  ├─ TechStatus struct
  │  ├─ TechnicianID: string     (16 bytes)
  │  ├─ TechnicianName: string   (32 bytes)
  │  ├─ CurrentBooking: string   (16 bytes)
  │  ├─ Status: string           (8 bytes)
  │  ├─ Latitude: float64        (8 bytes)
  │  ├─ Longitude: float64       (8 bytes)
  │  ├─ LastUpdate: int64        (8 bytes)
  │  ├─ Distance: float64        (8 bytes)
  │  └─ Overhead (struct/pointer)(40 bytes)
  │  Total: ~160 bytes
  │
  ├─ Map entry:                  (~100 bytes)
  │
  └─ LastReportTime entry:       (~100 bytes)

Per technician: ~360 bytes

100 technicians: 36 KB
1000 technicians: 360 KB
10000 technicians: 3.6 MB

Broker channels (estimate):
  ├─ Admin channel: 1 × 16 KB (pointers to clients)
  ├─ Customer channels: N × 8 KB (per booking)
  └─ Tech channels: N × 8 KB (per technician)

Total for 100 techs with 100 concurrent customers:
≈ 100 KB to 1 MB (very manageable)
```

---

## Throttling Mechanism

```
Update received at tech.js → POST /api/location/update

LocationCache.UpdateTechLocation()
{
    now := time.Now().UnixMilli()
    lastTime := lc.lastReportTime[techID]
    
    isNewUpdate := (now - lastTime) >= 10000  // 10 seconds
    
    if isNewUpdate {
        ✅ BROADCAST TO BROKER
        ├─ Admin channel
        ├─ Customer channel  
        └─ Tech channel
        
        lc.lastReportTime[techID] = now
    } else {
        ⏱️ SKIP BROADCAST
        └─ Just cache the position
        └─ Return "throttled" status
    }
}

Timeline Example:
─────────────────

t=0s:     Update received → lastTime = ?
          isNewUpdate = true (first)
          ✅ Broadcast #1
          lastTime = 0s

t=3s:     Update received → lastTime = 0s
          isNewUpdate = false (3s < 10s)
          ⏱️ Skip
          
t=7s:     Update received → lastTime = 0s
          isNewUpdate = false (7s < 10s)
          ⏱️ Skip

t=10s:    Update received → lastTime = 0s
          isNewUpdate = true (10s >= 10s) ✓
          ✅ Broadcast #2
          lastTime = 10s

t=15s:    Update received → lastTime = 10s
          isNewUpdate = false (5s < 10s)
          ⏱️ Skip

t=20s:    Update received → lastTime = 10s
          isNewUpdate = true (10s >= 10s) ✓
          ✅ Broadcast #3
          lastTime = 20s
```

---

These diagrams provide visual understanding of:
1. Overall system architecture
2. Message flow and timing
3. Event broadcasting mechanism
4. Client-side tracking flow
5. Geofence detection algorithm
6. Database schema
7. Memory management
8. Throttling strategy

Use these for documentation and team alignment!
