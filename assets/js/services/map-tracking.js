/**
 * Map Tracking Service
 * Handles real-time map updates with smooth marker movement
 * using Leaflet library
 * 
 * Usage:
 *   const mapTracker = new MapTracker(mapInstance);
 *   mapTracker.updateTechnicianLocation(techId, lat, lng);
 *   mapTracker.enableInterpolation(techId, duration);
 */

class MapTracker {
  constructor(leafletMap, options = {}) {
    this.map = leafletMap;
    
    // Configuration
    this.interpolationDuration = options.interpolationDuration || 8000; // 8 seconds smooth animation
    this.techMarkers = {}; // Store markers by tech_id
    this.techLines = {}; // Store polylines by tech_id
    this.interpolationFrameRate = options.frameRate || 30; // 30fps for smooth animation
    
    // Marker icons
    this.techMarkerIcon = this.createTechMarkerIcon();
    this.arrivalMarkerIcon = this.createArrivalMarkerIcon();
    this.customerMarkerIcon = this.createCustomerMarkerIcon();
    
    // Tracking state
    this.animationState = {};
    this.pathHistory = {}; // Store path history for drawing lines
    this.maxPathPoints = options.maxPathPoints || 50;
    
    // Callbacks
    this.onMarkerClick = options.onMarkerClick || (() => {});
    this.onArrived = options.onArrived || (() => {});
  }

  /**
   * Create Leaflet marker icon for technician
   */
  createTechMarkerIcon() {
    return L.divIcon({
      html: `
        <div class="tech-marker" style="
          width: 40px;
          height: 40px;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          border-radius: 50%;
          border: 3px solid white;
          box-shadow: 0 2px 8px rgba(0,0,0,0.3);
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
          font-size: 20px;
          font-weight: bold;
          position: relative;
        ">
          <span class="pulse-ring" style="
            position: absolute;
            width: 100%;
            height: 100%;
            border: 2px solid rgba(102, 126, 234, 0.4);
            border-radius: 50%;
            animation: pulse 2s infinite;
          "></span>
          👤
        </div>
      `,
      iconSize: [40, 40],
      iconAnchor: [20, 20],
      popupAnchor: [0, -20],
      className: 'tech-marker-icon'
    });
  }

  /**
   * Create marker icon for arrival notification
   */
  createArrivalMarkerIcon() {
    return L.divIcon({
      html: `
        <div class="arrival-marker" style="
          width: 50px;
          height: 50px;
          background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
          border-radius: 50%;
          border: 3px solid white;
          box-shadow: 0 4px 12px rgba(0,0,0,0.3);
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
          font-size: 24px;
          animation: bounce 0.6s infinite;
        ">
          ✓
        </div>
      `,
      iconSize: [50, 50],
      iconAnchor: [25, 25],
      popupAnchor: [0, -25],
      className: 'arrival-marker-icon'
    });
  }

  /**
   * Create marker icon for customer location
   */
  createCustomerMarkerIcon() {
    return L.divIcon({
      html: `
        <div class="customer-marker" style="
          width: 40px;
          height: 40px;
          background: #ff6b6b;
          border-radius: 0 50% 50% 0;
          border: 2px solid white;
          box-shadow: 0 2px 8px rgba(0,0,0,0.2);
          transform: rotate(-45deg);
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
          font-size: 18px;
        ">
          🏠
        </div>
      `,
      iconSize: [40, 40],
      iconAnchor: [20, 20],
      popupAnchor: [0, -20],
      className: 'customer-marker-icon'
    });
  }

  /**
   * Add or update technician location on map
   * If animation is in progress, interpolate smoothly to new position
   */
  updateTechnicianLocation(techId, techName, latitude, longitude, customerLat, customerLng, distance = null) {
    const newLatLng = [latitude, longitude];
    
    // If marker doesn't exist, create it
    if (!this.techMarkers[techId]) {
      this.createTechnicianMarker(techId, techName, newLatLng);
    }
    
    const marker = this.techMarkers[techId];
    
    // If we have an ongoing animation, cancel it and use the last animated position
    if (this.animationState[techId] && this.animationState[techId].animationId) {
      cancelAnimationFrame(this.animationState[techId].animationId);
    }
    
    // Store path history for visualization
    if (!this.pathHistory[techId]) {
      this.pathHistory[techId] = [];
    }
    this.pathHistory[techId].push(newLatLng);
    if (this.pathHistory[techId].length > this.maxPathPoints) {
      this.pathHistory[techId].shift();
    }
    
    // Update path line on map
    this.updatePathLine(techId);
    
    // Start smooth animation to new position
    this.animateMarkerToPosition(techId, marker, newLatLng);
    
    // Update info (distance, status, etc.)
    if (marker.distanceLabel && distance !== null) {
      marker.distanceLabel.setContent(this.formatDistance(distance));
    }
    
    // Update customer marker if provided
    if (customerLat && customerLng) {
      this.updateCustomerMarker(techId, customerLat, customerLng);
    }
    
    // Fit map to show both markers
    this.fitMapToMarkers(techId);
  }

  /**
   * Create initial technician marker
   */
  createTechnicianMarker(techId, techName, latlng) {
    const marker = L.marker(latlng, {
      icon: this.techMarkerIcon,
      draggable: false,
      title: techName
    }).addTo(this.map);
    
    // Add popup with tech info
    marker.bindPopup(`
      <div class="tech-popup" style="min-width: 150px;">
        <strong>${techName}</strong><br>
        <small>ID: ${techId}</small><br>
        <div id="distance-${techId}" style="margin-top: 5px; font-size: 12px;"></div>
      </div>
    `);
    
    // Add distance label
    const distanceLabel = L.tooltip({
      permanent: true,
      direction: 'right',
      offset: [20, 0],
      opacity: 0.8,
      className: 'distance-label'
    });
    marker.bindTooltip(distanceLabel);
    marker.distanceLabel = distanceLabel;
    
    // Add click handler
    marker.on('click', () => {
      this.onMarkerClick({ techId, techName, latlng });
    });
    
    this.techMarkers[techId] = marker;
    
    // Initialize animation state
    this.animationState[techId] = {
      currentLatLng: latlng,
      targetLatLng: latlng,
      progress: 1,
      animationId: null
    };
  }

  /**
   * Smoothly animate marker movement
   * Uses requestAnimationFrame for smooth 60fps animation
   */
  animateMarkerToPosition(techId, marker, targetLatLng) {
    const state = this.animationState[techId];
    if (!state) return;
    
    const startLatLng = state.currentLatLng || marker.getLatLng();
    state.currentLatLng = startLatLng;
    state.targetLatLng = targetLatLng;
    state.progress = 0;
    
    const startTime = performance.now();
    const duration = this.interpolationDuration;
    
    const animate = (currentTime) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / duration, 1);
      
      // Easing function: ease-in-out
      const eased = progress < 0.5 
        ? 2 * progress * progress 
        : -1 + 4 * progress - 2 * progress * progress;
      
      // Linear interpolation between current and target position
      const interpolatedLat = startLatLng[0] + (targetLatLng[0] - startLatLng[0]) * eased;
      const interpolatedLng = startLatLng[1] + (targetLatLng[1] - startLatLng[1]) * eased;
      
      marker.setLatLng([interpolatedLat, interpolatedLng]);
      state.currentLatLng = [interpolatedLat, interpolatedLng];
      state.progress = progress;
      
      if (progress < 1) {
        state.animationId = requestAnimationFrame(animate);
      } else {
        // Animation complete
        marker.setLatLng(targetLatLng);
        state.currentLatLng = targetLatLng;
        state.animationId = null;
      }
    };
    
    state.animationId = requestAnimationFrame(animate);
  }

  /**
   * Update path line (polyline) showing technician route
   */
  updatePathLine(techId) {
    // Remove old line
    if (this.techLines[techId]) {
      this.map.removeLayer(this.techLines[techId]);
    }
    
    // Create new line
    const pathPoints = this.pathHistory[techId];
    if (pathPoints.length > 1) {
      const polyline = L.polyline(pathPoints, {
        color: '#667eea',
        weight: 2,
        opacity: 0.5,
        smoothFactor: 1.0,
        dashArray: '5, 5'
      }).addTo(this.map);
      
      this.techLines[techId] = polyline;
    }
  }

  /**
   * Update customer location marker
   */
  updateCustomerMarker(techId, customerLat, customerLng) {
    const markerId = `customer-${techId}`;
    
    if (!this.techMarkers[markerId]) {
      const marker = L.marker([customerLat, customerLng], {
        icon: this.customerMarkerIcon,
        title: 'Địa chỉ khách hàng'
      }).addTo(this.map);
      
      marker.bindPopup('📍 Địa chỉ khách hàng');
      this.techMarkers[markerId] = marker;
    } else {
      this.techMarkers[markerId].setLatLng([customerLat, customerLng]);
    }
  }

  /**
   * Show arrival notification on map
   */
  showArrivalNotification(techId, techName, message) {
    const techMarker = this.techMarkers[techId];
    if (!techMarker) return;
    
    // Change marker icon
    techMarker.setIcon(this.arrivalMarkerIcon);
    
    // Update popup
    techMarker.bindPopup(`
      <div class="arrival-popup" style="min-width: 200px; text-align: center;">
        <strong style="color: #11998e;">✓ ${message}</strong><br>
        <small>${techName}</small>
      </div>
    `).openPopup();
    
    // Trigger callback
    this.onArrived({ techId, techName, message });
    
    // Animate back to tech icon after 5 seconds
    setTimeout(() => {
      if (techMarker && this.techMarkers[techId]) {
        techMarker.setIcon(this.techMarkerIcon);
      }
    }, 5000);
  }

  /**
   * Fit map bounds to show all relevant markers
   */
  fitMapToMarkers(techId) {
    const markers = [];
    
    if (this.techMarkers[techId]) {
      markers.push(this.techMarkers[techId]);
    }
    
    const customerMarkerId = `customer-${techId}`;
    if (this.techMarkers[customerMarkerId]) {
      markers.push(this.techMarkers[customerMarkerId]);
    }
    
    if (markers.length > 1) {
      const group = new L.featureGroup(markers);
      this.map.fitBounds(group.getBounds().pad(0.1));
    }
  }

  /**
   * Clear all markers and lines from map
   */
  clearAll() {
    Object.values(this.techMarkers).forEach(marker => {
      this.map.removeLayer(marker);
    });
    Object.values(this.techLines).forEach(line => {
      this.map.removeLayer(line);
    });
    
    this.techMarkers = {};
    this.techLines = {};
    this.animationState = {};
    this.pathHistory = {};
  }

  /**
   * Remove specific technician from map
   */
  removeTechnician(techId) {
    if (this.techMarkers[techId]) {
      this.map.removeLayer(this.techMarkers[techId]);
      delete this.techMarkers[techId];
    }
    
    const customerMarkerId = `customer-${techId}`;
    if (this.techMarkers[customerMarkerId]) {
      this.map.removeLayer(this.techMarkers[customerMarkerId]);
      delete this.techMarkers[customerMarkerId];
    }
    
    if (this.techLines[techId]) {
      this.map.removeLayer(this.techLines[techId]);
      delete this.techLines[techId];
    }
    
    if (this.animationState[techId]) {
      if (this.animationState[techId].animationId) {
        cancelAnimationFrame(this.animationState[techId].animationId);
      }
      delete this.animationState[techId];
    }
    
    if (this.pathHistory[techId]) {
      delete this.pathHistory[techId];
    }
  }

  /**
   * Format distance for display
   */
  formatDistance(distanceInMeters) {
    if (distanceInMeters < 1000) {
      return `${Math.round(distanceInMeters)}m`;
    } else {
      return `${(distanceInMeters / 1000).toFixed(1)}km`;
    }
  }

  /**
   * Get current state of all tracked technicians
   */
  getStatus() {
    const status = {};
    for (const [techId, state] of Object.entries(this.animationState)) {
      status[techId] = {
        position: state.currentLatLng,
        animating: state.animationId !== null,
        progress: state.progress
      };
    }
    return status;
  }

  /**
   * Enable/disable smooth interpolation
   */
  enableInterpolation(techId, enabled = true) {
    if (this.animationState[techId]) {
      this.animationState[techId].interpolationEnabled = enabled;
    }
  }
}

// Export for use
if (typeof module !== 'undefined' && module.exports) {
  module.exports = MapTracker;
}
