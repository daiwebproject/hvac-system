# Real-Time Tracking Frontend Integration Checklist

## Overview
Backend is 100% ready. Frontend needs integration with the new JavaScript services.

## Tech Dashboard Integration

### ✅ What's Ready (Backend)
- [x] Location API endpoints
- [x] In-memory cache for tech positions
- [x] Geofence detection (100m arrival)
- [x] Event broadcasting via SSE
- [x] Database updates (status, final location)

### ⏳ Items Remaining (Frontend)

#### 1. Tech Dashboard - Location Tracking UI
**File**: `views/pages/tech_dashboard.html`

```html
<!-- Add tracking controls -->
<div id="tracking-controls" class="tracking-panel" style="display: none;">
  <div class="tracking-status">
    <span id="tracking-distance" class="distance-badge">0 m</span>
    <span id="tracking-status" class="status-badge">Đang di chuyển...</span>
  </div>
  <button id="btn-stop-tracking" class="btn btn-danger">Dừng quá trình</button>
</div>

<!-- Add map container -->
<div id="tracking-map" style="height: 500px; display: none;"></div>

<!-- Include scripts -->
<script src="/assets/js/services/location-tracking.js"></script>
<script src="/assets/js/services/map-tracking.js"></script>
<script src="/assets/vendor/leaflet/leaflet.js"></script>
<script>
  // Load when component is ready
  if (window.techCurrentJob) {
    initializeTracking(window.techCurrentJob);
  }
</script>
```

**Implementation**:
- [ ] Get current booking ID from page
- [ ] Initialize LocationTracker with booking ID
- [ ] Request geolocation permissions
- [ ] Show tracking UI when job status = "moving"
- [ ] Update distance display in real-time
- [ ] Show map with markers
- [ ] Display arrival notification when reached

#### 2. Tech Job Detail View
**File**: `views/pages/tech_job_detail.html`

Add "Bắt đầu di chuyển" button:
```html
<button id="btn-start-moving" class="btn btn-primary" hx-post="/api/tech/tracking/start">
  Bắt đầu di chuyển →
</button>
```

Add completion handler:
```javascript
document.getElementById('btn-complete-job').addEventListener('click', async (e) => {
  e.preventDefault();
  
  // Stop tracking first
  if (window.locationTracker && window.locationTracker.isTracking) {
    await window.locationTracker.stopTracking();
  }
  
  // Then submit job completion
  // ... existing completion logic
});
```

**Implementation**:
- [ ] Button to start tracking
- [ ] Show tracking status while moving
- [ ] Disable "Hoàn thành" until arrival (optional)
- [ ] Stop tracking on job completion

#### 3. Admin Dashboard - Location Stream
**File**: `views/pages/admin_dashboard.html`

```html
<!-- Add map for tech tracking -->
<div id="admin-map" class="admin-section">
  <h2>Bản đồ theo dõi</h2>
  <div id="map" style="height: 600px;"></div>
  <div id="tech-list" style="max-height: 300px; overflow-y: auto;">
    <!-- Tech cards with location -->
  </div>
</div>

<script>
  const mapElement = document.getElementById('map');
  if (mapElement) {
    initializeAdminTracking();
  }
</script>
```

**Implementation**:
- [ ] Initialize Leaflet map focused on city
- [ ] Connect SSE stream for all techs
- [ ] Show tech markers with icons
- [ ] Display distance to their bookings
- [ ] Update in real-time as techs move
- [ ] Show arrival notifications
- [ ] Add tech list sidebar

#### 4. Customer/Public Booking Tracking
**File**: `views/pages/public_booking_tracking.html` (New)

```html
<div class="booking-tracking">
  <h1>Theo dõi thợ của tôi</h1>
  
  <!-- Status display -->
  <div id="tech-status-card" class="status-card">
    <div class="tech-name">{{ technician_name }}</div>
    <div class="distance-info">
      <span id="distance">Đang tìm kiếm vị trí...</span>
      <span id="eta">ETA: --</span>
    </div>
  </div>
  
  <!-- Live map -->
  <div id="map" style="height: 500px;"></div>
  
  <!-- Arrival notification -->
  <div id="arrival-notification" style="display: none;" class="alert alert-success">
    ✓ Thợ đã đến cổng nhà bạn!
  </div>
</div>

<script src="/assets/js/services/location-tracking.js"></script>
<script src="/assets/js/services/map-tracking.js"></script>
<script>
  const bookingId = '{{ booking_id }}';
  const customerLat = {{ customer_lat }};
  const customerLng = {{ customer_lng }};
  
  initializeCustomerTracking(bookingId, customerLat, customerLng);
</script>
```

**Implementation**:
- [ ] Show customer location (home icon)
- [ ] Show tech location and marker
- [ ] Display distance in real-time
- [ ] Show ETA based on speed/distance
- [ ] SSE stream for live updates
- [ ] Arrival notification when <100m
- [ ] Handle offline gracefully

#### 5. Additional Views to Update

**Views/Partials to Check**:
- [ ] `status_stepper.html` - Add "Đang di chuyển" indicator
- [ ] `job_actions_bar.html` - Add tracking button
- [ ] `tech/dashboard_stats.html` - Show location updates
- [ ] `tech/partials/jobs_list.html` - Show distance for active job

## CSS Styling

**Create/Update**: `assets/css/tracking.css`

```css
/* Tracking UI */
.tracking-panel {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 15px;
  border-radius: 8px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 15px;
}

.distance-badge {
  font-size: 24px;
  font-weight: bold;
  font-variant-numeric: tabular-nums;
}

.status-badge {
  font-size: 14px;
  opacity: 0.8;
}

/* Map styling */
#map, #admin-map, #tracking-map {
  border: 1px solid #ddd;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0,0,0,0.1);
}

/* Tech marker animation */
@keyframes pulse {
  0%, 100% {
    transform: scale(1);
    opacity: 1;
  }
  50% {
    transform: scale(1.1);
    opacity: 0.8;
  }
}

.tech-marker-icon {
  animation: pulse 2s infinite;
}

@keyframes bounce {
  0%, 100% {
    transform: translateY(0);
  }
  50% {
    transform: translateY(-10px);
  }
}

.arrival-marker-icon {
  animation: bounce 0.6s infinite;
}

/* Status card */
.status-card {
  background: white;
  border: 1px solid #ddd;
  border-radius: 8px;
  padding: 15px;
  margin-bottom: 15px;
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.tech-name {
  font-size: 18px;
  font-weight: 600;
}

.distance-info {
  display: flex;
  gap: 20px;
  text-align: right;
}

.distance-info span {
  display: block;
  font-size: 14px;
  color: #666;
}

/* Arrival notification */
.arrival-notification {
  animation: slideDown 0.3s ease;
  position: sticky;
  top: 0;
  z-index: 100;
}

@keyframes slideDown {
  from {
    transform: translateY(-100%);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}

/* Loading state */
.tracking-loading {
  opacity: 0.6;
  cursor: wait;
}
```

## JavaScript Helper Functions

**Create**: `assets/js/tracking-integration.js`

```javascript
/**
 * Initialize tech dashboard tracking
 */
async function initializeTracking(bookingData) {
  // Request permission
  if (!LocationTracker.isSupported()) {
    alert('Thiết bị không hỗ trợ định vị');
    return;
  }
  
  const hasPermission = await LocationTracker.requestPermission();
  if (!hasPermission) {
    alert('Vui lòng bật quyền GPS');
    return;
  }
  
  // Create tracker
  window.locationTracker = new LocationTracker(
    bookingData.technician_id,
    bookingData.booking_id,
    {
      onLocationUpdate: (data) => {
        updateTrackerUI(data);
      },
      onArrived: (data) => {
        showArrivalAlert(data);
      }
    }
  );
  
  showTrackerUI();
}

/**
 * Update tracker UI with location data
 */
function updateTrackerUI(data) {
  if (!data.success) return;
  
  const distance = data.response.distance;
  const eta = calculateETA(distance, data.speed);
  
  document.getElementById('tracking-distance').textContent = 
    formatDistance(distance);
  document.getElementById('tracking-eta').textContent = 
    `ETA: ${eta}`;
  
  // Update map if visible
  if (window.mapTracker) {
    window.mapTracker.updateTechnicianLocation(
      data.technician_id,
      data.technician_name,
      data.latitude,
      data.longitude,
      bookingData.customer_lat,
      bookingData.customer_lng,
      distance
    );
  }
}

/**
 * Initialize admin location tracking
 */
function initializeAdminTracking() {
  // Leaflet map
  const map = L.map('admin-map').setView([21.0285, 105.8542], 13);
  L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
    attribution: '© OpenStreetMap',
    maxZoom: 19
  }).addTo(map);
  
  window.mapTracker = new MapTracker(map);
  
  // Connect SSE stream
  const source = new EventSource('/api/admin/api/locations/stream');
  
  source.addEventListener('location.updated', (event) => {
    const data = JSON.parse(event.data).data;
    window.mapTracker.updateTechnicianLocation(
      data.technician_id,
      data.technician_name,
      data.latitude,
      data.longitude,
      null, null,
      data.distance
    );
    updateTechListUI(data);
  });
  
  source.addEventListener('geofence.arrived', (event) => {
    const data = JSON.parse(event.data).data;
    window.mapTracker.showArrivalNotification(
      data.technician_id,
      data.technician_name,
      data.message
    );
  });
}

/**
 * Utility functions
 */
function formatDistance(meters) {
  if (meters < 1000) return Math.round(meters) + ' m';
  return (meters / 1000).toFixed(1) + ' km';
}

function calculateETA(distanceMeters, speedMs) {
  if (speedMs < 0.5) return '...';
  const minutes = Math.ceil(distanceMeters / (speedMs * 60));
  return `${minutes} phút`;
}

function showArrivalAlert(data) {
  // Show notification
  const notification = document.getElementById('arrival-notification');
  if (notification) {
    notification.style.display = 'block';
    notification.textContent = '✓ ' + (data.message || 'Thợ đã đến nơi');
  }
}
```

## Integration Steps

1. **Add Scripts to Base Template**
   ```html
   <script src="/assets/js/services/location-tracking.js"></script>
   <script src="/assets/js/services/map-tracking.js"></script>
   <script src="/assets/js/tracking-integration.js"></script>
   <link rel="stylesheet" href="/assets/css/tracking.css">
   ```

2. **Update Tech Dashboard**
   - Add map div
   - Add tracking controls
   - Initialize on page load

3. **Update Admin Dashboard**
   - Add map for all techs
   - Initialize SSE stream
   - Update tech list

4. **Create Customer Tracking Page**
   - Get booking details
   - Initialize map
   - Stream tech location

5. **Test All Flows**
   - Tech can start/stop tracking
   - Admin sees all techs
   - Customer sees their tech
   - Arrival notification works

## Error Scenarios to Handle

- [ ] GPS permission denied → Show user-friendly message
- [ ] GPS signal lost → Show "(Signal lost)" state
- [ ] Network error → Queue updates, retry
- [ ] SSE disconnected → Auto-reconnect after delay
- [ ] Browser doesn't support geolocation → Show warning
- [ ] Phone goes into power save → Respect device settings

## Performance Considerations

- [ ] Load map library only when needed
- [ ] Unload trackers when leaving page
- [ ] Clean up SSE connections
- [ ] Don't track if user minimizes app
- [ ] Respect device battery saver mode

## Browser Compatibility

- [x] Chrome 50+
- [x] Firefox 53+
- [x] Safari 12+
- [x] Edge 15+
- [x] Opera 37+
- ⚠️ IE 11 - Not supported (use polyfills if needed)

## Testing Before Production

```bash
# 1. Start backend
go run main.go serve

# 2. Open two tabs
# Tab 1: Tech dashboard (http://localhost:8090/tech)
# Tab 2: Admin dashboard (http://localhost:8090/admin)

# 3. Tech clicks "Bắt đầu di chuyển"
# 4. Verify admin map updates in real-time
# 5. Test arrival notification at <100m

# 6. Open customer tracking page
# 7. Verify customer sees tech approaching
# 8. Check performance with multiple techs
```

---

**Status**: Backend 100% complete, frontend ready for integration
**Next Steps**: Implement views and test with real data
