/**
 * Tech Dashboard - Location Tracking Integration
 * 
 * This is a complete example of how to integrate location tracking
 * into the technician dashboard.
 * 
 * In your tech.html template, add:
 *   <script src="/assets/js/services/location-tracking.js"></script>
 *   <button @click="startTracking()" x-text="isTracking ? 'Dừng' : 'Bắt đầu di chuyển'"></button>
 */

window.techLocationTracking = function (initData) {
  return {
    isTracking: false,
    isOnline: navigator.onLine,
    currentBooking: initData?.bookingId || null,
    currentTech: initData?.techId || null,
    status: initData?.jobStatus || null,

    // Tracker instance
    tracker: null,

    // Status UI
    trackingStatus: 'stopped',
    lastLocationTime: null,
    batteryLevel: 100,
    errorMessage: null,
    successMessage: null,

    // Stats
    sentLocations: 0,
    failedCount: 0,

    init() {
      console.log('🚀 Tech Location Tracking Initialized', this.status);

      // Request location permission on init
      this.requestLocationPermission();

      // Listen for online/offline events
      window.addEventListener('online', () => {
        this.isOnline = true;
        this.clearError();
      });
      window.addEventListener('offline', () => {
        this.isOnline = false;
        this.setError('Mất kết nối Internet - vị trí sẽ không được cập nhật');
      });

      // Monitor battery level (optional)
      this.monitorBattery();

      // Auto-start tracking if status is accepted or moving
      if (['accepted', 'moving'].includes(this.status)) {
        console.log('🔄 Auto-starting tracking based on status:', this.status);
        this.startTracking();
      }
    },

    /**
     * Start location tracking
     */
    async startTracking() {
      if (this.isTracking) {
        console.warn('Already tracking');
        return;
      }

      if (!this.isOnline) {
        this.setError('Không có kết nối Internet. Vui lòng kiểm tra mạng 3G/4G hoặc WiFi.');
        return;
      }

      if (!this.currentBooking || !this.currentTech) {
        this.setError('Thông tin đơn hàng hoặc thợ bị thiếu');
        return;
      }

      try {
        // Initialize tracker if not already done
        if (!this.tracker) {
          this.tracker = new LocationTracker(
            this.currentTech,
            this.currentBooking,
            {
              throttleInterval: 10000, // 10 seconds
              highAccuracyMode: true,
              timeout: 10000,
              maxAge: 5000,
              apiEndpoint: '/api/tech/location/update',

              // Callbacks
              onLocationUpdate: (data) => this.handleLocationUpdate(data),
              onArrived: (data) => this.handleArrived(data),
              onError: (error) => this.handleTrackingError(error),
              onStatusChange: (status) => this.handleStatusChange(status)
            }
          );
        }

        // Start tracking
        const success = await this.tracker.startTracking();
        if (success) {
          this.isTracking = true;
          this.trackingStatus = 'tracking';
          this.clearError();
          this.setSuccess('Đã bắt đầu theo dõi vị trí');
        }

      } catch (error) {
        this.setError(`Lỗi: ${error.message}`);
      }
    },

    /**
     * Stop tracking
     */
    async stopTracking() {
      if (!this.isTracking || !this.tracker) {
        return;
      }

      try {
        const success = await this.tracker.stopTracking();
        if (success) {
          this.isTracking = false;
          this.trackingStatus = 'stopped';
          this.setSuccess('Đã dừng theo dõi vị trí');
        }
      } catch (error) {
        this.setError(`Lỗi khi dừng: ${error.message}`);
      }
    },

    /**
     * Handle location update from tracker
     */
    handleLocationUpdate(data) {
      if (data.throttled) {
        // Only throttled, don't update UI too much
        return;
      }

      if (data.success) {
        this.sentLocations++;
        this.lastLocationTime = new Date().toLocaleTimeString('vi-VN');

        if (data.response?.distance !== undefined) {
          // Update distance to customer in UI
          const distEl = document.getElementById('distance-to-customer');
          if (distEl) {
            distEl.textContent = `${this.formatDistance(data.response.distance)}`;
          }
        }
      } else {
        this.failedCount++;
      }

      console.log(`📍 Location update: ${data.latitude.toFixed(4)}, ${data.longitude.toFixed(4)}`);
    },

    /**
     * Handle arrival notification
     */
    handleArrived(data) {
      console.log('✨ Arrived at customer location!', data);

      // Show toast notification
      if (window.toast) {
        window.toast.show({
          message: data.message || 'Bạn đã đến địa chỉ khách hàng',
          type: 'success',
          duration: 5000
        });
      }

      // Update UI to show "Check In" button
      this.trackingStatus = 'arrived';
      document.getElementById('check-in-button')?.classList.remove('hidden');
    },

    /**
     * Handle tracking errors
     */
    handleTrackingError(error) {
      console.error('❌ Tracking error:', error);

      if (error.code === error.PERMISSION_DENIED) {
        this.setError('Quyền GPS bị từ chối. Vui lòng bật GPS trong cài đặt điện thoại.');
        this.stopTracking();
      } else if (error.type === 'send_failed') {
        this.failedCount++;
        // Don't show error every time, just count
      } else {
        this.setError(`Lỗi: ${error.message}`);
      }
    },

    /**
     * Handle status change
     */
    handleStatusChange(status) {
      console.log('📌 Status:', status);
      this.trackingStatus = status.isTracking ? 'tracking' : 'stopped';
    },

    /**
     * Request location permission from user
     */
    async requestLocationPermission() {
      if (!LocationTracker.isSupported()) {
        this.setError('Trình duyệt này không hỗ trợ định vị GPS');
        return false;
      }

      const granted = await LocationTracker.requestPermission();
      if (!granted) {
        this.setError('Vui lòng bật quyền GPS để sử dụng tính năng này');
      }
      return granted;
    },

    /**
     * Monitor device battery level
     */
    monitorBattery() {
      if ('getBattery' in navigator || 'battery' in navigator) {
        const battery = navigator.getBattery?.() || navigator.battery;
        if (battery) {
          battery.addEventListener?.('levelchange', () => {
            this.batteryLevel = Math.round(battery.level * 100);
          });
        }
      }
    },

    /**
     * Get current tracking status
     */
    getStatus() {
      if (!this.tracker) return null;
      return this.tracker.getStatus();
    },

    /**
     * Format distance for display
     */
    formatDistance(meters) {
      if (meters < 1000) {
        return `${Math.round(meters)}m`;
      } else {
        return `${(meters / 1000).toFixed(1)}km`;
      }
    },

    // UI Helpers
    setError(message) {
      this.errorMessage = message;
      this.successMessage = null;
      console.error(message);
    },

    setSuccess(message) {
      this.successMessage = message;
      this.errorMessage = null;
      console.log(message);
      setTimeout(() => {
        this.successMessage = null;
      }, 3000);
    },

    clearError() {
      this.errorMessage = null;
    },

    // Tracking status badges
    getStatusBadge() {
      switch (this.trackingStatus) {
        case 'tracking':
          return { text: '🟢 Đang theo dõi...', color: 'green' };
        case 'arrived':
          return { text: '🟡 Đã đến nơi', color: 'yellow' };
        case 'stopped':
          return { text: '🔴 Không theo dõi', color: 'gray' };
        default:
          return { text: '❓ Không xác định', color: 'gray' };
      }
    }
  };
};

// ============ ADMIN DASHBOARD INTEGRATION ============

/**
 * Admin Dashboard - Real-time Location Monitoring
 * 
 * Shows all technicians' locations on a map and updates in real-time via SSE
 */
window.adminLocationMonitoring = function (initData) {
  return {
    map: null,
    mapTracker: null,
    activeSSEConnection: false,
    technicians: new Map(), // Map[tech_id] -> TechStatus

    init() {
      console.log('🗺️ Admin Location Monitoring Initialized');

      // Expose instance for other components
      window.adminMapComponent = this;

      // Initialize map
      this.initializeMap();

      // Connect to SSE for real-time updates
      this.connectToLocationSSE();

      // Refresh location data every 30 seconds as fallback
      setInterval(() => this.refreshAllLocations(), 30000);
    },

    /**
     * Initialize Leaflet map
     */
    initializeMap() {
      const mapElement = document.getElementById('admin-map');
      if (!mapElement) return;

      // Initialize Leaflet map (adjust coordinates as needed)
      this.map = L.map('admin-map').setView([21.0285, 105.8542], 13);

      // Add tile layer
      L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 19
      }).addTo(this.map);

      // Initialize map tracker
      this.mapTracker = new MapTracker(this.map, {
        interpolationDuration: 8000,
        frameRate: 30,

        onMarkerClick: (marker) => this.handleMarkerClick(marker),
        onArrived: (data) => this.handleArrived(data)
      });

      console.log('✅ Map initialized');
    },

    /**
     * Connect to SSE stream for location updates
     */
    connectToLocationSSE() {
      // SSE endpoint via htmx-sse
      const eventSource = new EventSource('/admin/locations/stream');

      eventSource.addEventListener('location.updated', (event) => {
        console.log('Update received:', event.data);
        const data = JSON.parse(event.data);
        this.handleLocationUpdate(data);
      });

      eventSource.addEventListener('geofence.arrived', (event) => {
        const data = JSON.parse(event.data);
        this.mapTracker?.showArrivalNotification(
          data.technician_id,
          data.technician_name,
          data.message
        );
      });

      // [NEW] Handle Status Changes
      eventSource.addEventListener('job.status_changed', (event) => {
        const data = JSON.parse(event.data);
        console.log('Job Status Changed:', data);
        this.handleStatusUpdate(data.tech_id, data.status);
      });

      eventSource.addEventListener('tech.status_changed', (event) => {
        const data = JSON.parse(event.data);
        console.log('Tech Status Changed:', data);
        this.handleStatusUpdate(data.id, data.active ? 'online' : 'offline');
      });

      eventSource.addEventListener('error', (error) => {
        console.error('SSE error:', error);
        this.activeSSEConnection = false;
        // Fallback to polling
        setTimeout(() => this.connectToLocationSSE(), 5000);
      });

      eventSource.addEventListener('open', () => {
        console.log('✅ SSE Connected');
        this.activeSSEConnection = true;
      });
    },

    /**
     * Handle location update from SSE or API
     */
    handleLocationUpdate(data) {
      if (!data || !data.technician_id) return;

      // Store tech info
      this.technicians.set(data.technician_id, {
        id: data.technician_id,
        name: data.technician_name,
        booking: data.booking_id,
        lat: data.latitude,
        lng: data.longitude,
        distance: data.distance,
        lastUpdate: data.timestamp
      });

      // Update map marker
      const allTechs = Array.from(this.technicians.values());
      const tech = allTechs.find(t => t.id === data.technician_id);

      if (tech) {
        this.mapTracker?.updateTechnicianLocation(
          tech.id,
          tech.name,
          tech.lat,
          tech.lng,
          null, // customer lat
          null, // customer lng
          tech.distance
        );

        // Dispatch global event for other components (e.g. Kanban Map)
        document.dispatchEvent(new CustomEvent('admin:location-updated', { detail: tech }));
      }
    },

    /**
     * Handle status update (Job or Tech status)
     */
    handleStatusUpdate(techId, status) {
      if (!techId) return;

      // Update local state
      if (this.technicians.has(techId)) {
        const tech = this.technicians.get(techId);
        tech.status = status; // Add status field
        this.technicians.set(techId, tech);

        // Dispatch global event
        document.dispatchEvent(new CustomEvent('admin:location-updated', { detail: tech }));
      }

      // Update Map Marker
      if (this.mapTracker) {
        this.mapTracker.updateTechnicianStatus(techId, status);
      }
    },

    /**
     * Refresh all technician locations from API
     */
    async refreshAllLocations() {
      try {
        const response = await fetch('/admin/api/locations');
        if (!response.ok) return;

        const data = await response.json();
        if (data.techs) {
          data.techs.forEach(tech => {
            this.handleLocationUpdate({
              technician_id: tech.technician_id,
              technician_name: tech.technician_name,
              booking_id: tech.current_booking,
              latitude: tech.latitude,
              longitude: tech.longitude,
              distance: tech.distance,
              timestamp: tech.last_update
            });
          });
        }
      } catch (error) {
        console.error('Failed to refresh locations:', error);
      }
    },

    /**
     * Handle marker click
     */
    handleMarkerClick(marker) {
      const tech = this.technicians.get(marker.techId);
      if (!tech) return;

      console.log('Technician selected:', tech);

      // Trigger detail view or modal
      const event = new CustomEvent('tech-selected', { detail: tech });
      document.dispatchEvent(event);
    },

    /**
     * Handle arrival event
     */
    handleArrived(data) {
      // Show notification
      if ('Notification' in window && Notification.permission === 'granted') {
        new Notification('Thợ đã đến', {
          body: `${data.techName} đã đến địa chỉ khách hàng`,
          icon: '/assets/icons/location-arrived.png'
        });
      }
    },

    /**
     * Fit map to show all markers
     */
    fitBounds() {
      if (this.mapTracker) {
        this.mapTracker.fitMapToAllMarkers();
      }
    },

    /**
     * Get list of all active technicians
     */
    getActiveTechnicians() {
      return Array.from(this.technicians.values());
    }
  };
};
