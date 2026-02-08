/**
 * Location Tracking Service
 * Handles real-time location updates from technician device
 * 
 * Usage:
 *   const tracker = new LocationTracker(technicianId, bookingId);
 *   tracker.startTracking();
 *   tracker.stopTracking();
 */

class LocationTracker {
  constructor(technicianId, bookingId, options = {}) {
    this.technicianId = technicianId;
    this.bookingId = bookingId;

    // Configuration
    this.throttleInterval = options.throttleInterval || 10000; // 10 seconds
    this.highAccuracyMode = options.highAccuracyMode !== false; // Default true
    this.maxAge = options.maxAge || 5000; // 5 seconds max age
    this.timeout = options.timeout || 10000; // 10 seconds timeout
    this.geofenceRadius = options.geofenceRadius || 100; // 100 meters
    this.apiEndpoint = options.apiEndpoint || '/api/location';

    // State
    this.isTracking = false;
    this.watchId = null;
    this.lastSentTime = 0;
    this.lastLatitude = null;
    this.lastLongitude = null;
    this.batteryLevel = 100;

    // Stats
    this.sentCount = 0;
    this.errorCount = 0;
    this.pendingRequests = 0;

    // Callbacks
    this.onLocationUpdate = options.onLocationUpdate || (() => { });
    this.onArrived = options.onArrived || (() => { });
    this.onError = options.onError || (() => { });
    this.onStatusChange = options.onStatusChange || (() => { });
  }

  /**
   * Start tracking location
   * Triggers "Bắt đầu di chuyển" on backend
   */
  async startTracking() {
    if (this.isTracking) {
      console.log('⚠️ Location tracking already running');
      return false;
    }

    try {
      // First, notify backend that tracking is starting
      await this.notifyBackendStart();

      this.isTracking = true;
      this.lastSentTime = 0; // Allow immediate first update

      // Start watching position
      // watchPosition only fires when position actually changes
      // More battery-efficient than setInterval polling
      const geoOptions = {
        enableHighAccuracy: this.highAccuracyMode,
        timeline: this.timeout,
        maximumAge: this.maxAge
      };

      this.watchId = navigator.geolocation.watchPosition(
        (position) => this.handlePositionUpdate(position),
        (error) => this.handlePositionError(error),
        geoOptions
      );

      this.onStatusChange({ isTracking: true, message: 'Bắt đầu theo dõi vị trí' });
      console.log('✅ Location tracking started');

      return true;
    } catch (error) {
      console.error('❌ Failed to start tracking:', error);
      this.onError(error);
      return false;
    }
  }

  /**
   * Stop tracking location
   * Called when job is completed
   */
  async stopTracking() {
    if (!this.isTracking) {
      return false;
    }

    this.isTracking = false;

    // Clear watch
    if (this.watchId !== null) {
      navigator.geolocation.clearWatch(this.watchId);
      this.watchId = null;
    }

    try {
      // Notify backend that tracking stopped
      await this.notifyBackendStop();

      this.onStatusChange({ isTracking: false, message: 'Dừng theo dõi vị trí' });
      console.log('✅ Location tracking stopped');

      return true;
    } catch (error) {
      console.error('⚠️ Failed to notify stop:', error);
      return false;
    }
  }

  /**
   * Handle successful position update
   * Implements throttling to avoid too many API calls
   */
  handlePositionUpdate(position) {
    const { latitude, longitude, accuracy, speed, heading } = position.coords;
    const timestamp = position.timestamp;

    // Update cached position
    this.lastLatitude = latitude;
    this.lastLongitude = longitude;

    // Throttling check: only send if enough time has passed
    const now = Date.now();
    if (now - this.lastSentTime < this.throttleInterval) {
      console.log(`⏱️ Location throttled (${Math.round((this.throttleInterval - (now - this.lastSentTime)) / 1000)}s remaining)`);
      this.onLocationUpdate({
        latitude,
        longitude,
        accuracy,
        speed,
        heading,
        throttled: true
      });
      return;
    }

    // Send location to server
    this.sendLocationToServer({
      latitude,
      longitude,
      accuracy,
      speed: speed || 0,
      heading: heading || 0,
      timestamp
    });
  }

  /**
   * Send location to backend
   */
  async sendLocationToServer(locationData) {
    if (this.pendingRequests > 3) {
      console.warn('⚠️ Too many pending requests, skipping this update');
      return;
    }

    this.lastSentTime = Date.now();
    this.pendingRequests++;

    try {
      const payload = {
        technician_id: this.technicianId,
        booking_id: this.bookingId,
        latitude: locationData.latitude,
        longitude: locationData.longitude,
        accuracy: locationData.accuracy,
        speed: locationData.speed,
        heading: locationData.heading
      };

      const response = await fetch(this.apiEndpoint, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(payload)
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      const data = await response.json();
      this.sentCount++;

      console.log(`📍 Location sent (${this.sentCount} total)`, data);

      // Check if arrived
      if (data.arrived) {
        this.handleArrived(data);
      }

      this.onLocationUpdate({
        ...locationData,
        response: data,
        success: true
      });

    } catch (error) {
      this.errorCount++;
      console.error('❌ Failed to send location:', error);
      this.onError({
        type: 'send_failed',
        message: error.message,
        details: locationData
      });

      // Will retry on next position change (due to watchPosition)
    } finally {
      this.pendingRequests--;
    }
  }

  /**
   * Handle position error (denied, unavailable, timeout)
   */
  handlePositionError(error) {
    let message = 'Unknown error';

    switch (error.code) {
      case error.PERMISSION_DENIED:
        message = 'Quyền truy cập vị trí bị từ chối. Vui lòng bật GPS.';
        this.stopTracking(); // Stop if user denies
        break;
      case error.POSITION_UNAVAILABLE:
        message = 'Không thể xác định vị trí hiện tại.';
        break;
      case error.TIMEOUT:
        message = 'Quá thời gian yêu cầu vị trí.';
        break;
    }

    console.error(`❌ Geolocation error: ${message}`);
    this.onError({
      type: 'geolocation_error',
      code: error.code,
      message: message
    });
  }

  /**
   * Called when technician has arrived (server-side detection)
   */
  handleArrived(data) {
    console.log('✨ Technician has ARRIVED!', data);

    // Show notification
    if ('Notification' in window && Notification.permission === 'granted') {
      new Notification('Đã đến nơi', {
        body: 'Bạn đã đến địa chỉ khách hàng. Vui lòng bấm "Bắt đầu công việc" để tiếp tục.',
        icon: '/assets/icons/location-arrived.png',
        tag: 'arrival-notification'
      });
    }

    this.onArrived(data);
  }

  /**
   * Notify backend that tracking is starting
   */
  async notifyBackendStart() {
    try {
      const response = await fetch('/api/tech/tracking/start', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: new URLSearchParams({
          technician_id: this.technicianId,
          booking_id: this.bookingId
        })
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      console.log('✅ Backend notified: tracking started');
    } catch (error) {
      console.error('⚠️ Failed to notify backend start:', error);
      throw error;
    }
  }

  /**
   * Notify backend that tracking has stopped
   */
  async notifyBackendStop() {
    try {
      const response = await fetch('/api/tech/tracking/stop', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded'
        },
        body: new URLSearchParams({
          technician_id: this.technicianId,
          booking_id: this.bookingId
        })
      });

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}`);
      }

      console.log('✅ Backend notified: tracking stopped');
    } catch (error) {
      console.error('⚠️ Failed to notify backend stop:', error);
      throw error;
    }
  }

  /**
   * Get current tracking status
   */
  getStatus() {
    return {
      isTracking: this.isTracking,
      technicianId: this.technicianId,
      bookingId: this.bookingId,
      lastLocation: {
        latitude: this.lastLatitude,
        longitude: this.lastLongitude
      },
      stats: {
        sentCount: this.sentCount,
        errorCount: this.errorCount,
        pendingRequests: this.pendingRequests
      }
    };
  }

  /**
   * Get permission from user for location access
   */
  static async requestPermission() {
    if (!('geolocation' in navigator)) {
      console.error('❌ Geolocation not supported in this browser');
      return false;
    }

    // Request high level of accuracy
    return new Promise((resolve) => {
      navigator.geolocation.getCurrentPosition(
        () => {
          console.log('✅ Location permission granted');
          resolve(true);
        },
        (error) => {
          console.error('❌ Location permission denied:', error.message);
          resolve(false);
        },
        { enableHighAccuracy: true, timeout: 10000 }
      );
    });
  }

  /**
   * Check if geolocation is available
   */
  static isSupported() {
    return 'geolocation' in navigator;
  }

  /**
   * Calculate distance between two coordinates (Haversine formula)
   */
  static calculateDistance(lat1, lng1, lat2, lng2) {
    const R = 6371000; // Earth radius in meters
    const dLat = (lat2 - lat1) * Math.PI / 180;
    const dLng = (lng2 - lng1) * Math.PI / 180;
    const a =
      Math.sin(dLat / 2) * Math.sin(dLat / 2) +
      Math.cos(lat1 * Math.PI / 180) * Math.cos(lat2 * Math.PI / 180) *
      Math.sin(dLng / 2) * Math.sin(dLng / 2);
    const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
    return R * c;
  }
}

// Export for use in other scripts
if (typeof module !== 'undefined' && module.exports) {
  module.exports = LocationTracker;
}
