package handler

import (
	"encoding/json"
	"fmt"
	"hvac-system/pkg/broker"
	"log"
	"time"

	pbCore "github.com/pocketbase/pocketbase/core"
)

type LocationSSEHandler struct {
	broker *broker.SegmentedBroker
}

func NewLocationSSEHandler(broker *broker.SegmentedBroker) *LocationSSEHandler {
	return &LocationSSEHandler{
		broker: broker,
	}
}

// StreamAdminLocations handles GET /api/admin/locations/stream
// SSE stream for admin dashboard to receive all technician locations in real-time
func (h *LocationSSEHandler) StreamAdminLocations(e *pbCore.RequestEvent) error {
	// Set SSE headers
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to admin channel
	eventChan := h.broker.Subscribe(broker.ChannelAdmin, "")
	defer h.broker.Unsubscribe(broker.ChannelAdmin, "", eventChan)

	// Send initial connection message
	sendSSEMessage(e, "connected", map[string]interface{}{
		"message":   "Connected to location stream",
		"timestamp": time.Now().Unix(),
	})

	// Listen for events
	for event := range eventChan {
		// Only send location-related events
		if event.Type == "location.updated" || event.Type == "geofence.arrived" || 
		   event.Type == "tracking.started" || event.Type == "tracking.stopped" {
			sendSSEMessage(e, event.Type, event.Data)
		}
	}

	return nil
}

// StreamCustomerLocation handles GET /api/bookings/{id}/location/stream
// SSE stream for customer to receive their technician's location in real-time
func (h *LocationSSEHandler) StreamCustomerLocation(e *pbCore.RequestEvent) error {
	bookingID := e.Request.PathValue("id")
	if bookingID == "" {
		return e.JSON(400, map[string]string{"error": "Missing booking ID"})
	}

	// Set SSE headers
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to customer channel for this booking
	eventChan := h.broker.Subscribe(broker.ChannelCustomer, bookingID)
	defer h.broker.Unsubscribe(broker.ChannelCustomer, bookingID, eventChan)

	// Send initial connection message
	sendSSEMessage(e, "connected", map[string]interface{}{
		"message":    "Connected to technician tracking",
		"booking_id": bookingID,
		"timestamp":  time.Now().Unix(),
	})

	// Listen for events
	for event := range eventChan {
		// Send all events related to this booking
		if event.Type == "location.updated" || event.Type == "geofence.arrived" || 
		   event.Type == "tracking.started" || event.Type == "tracking.stopped" {
			sendSSEMessage(e, event.Type, event.Data)
		}
	}

	return nil
}

// StreamTechnicianEvents handles GET /api/tech/:id/events/stream
// SSE stream for technician to receive job assignments and status updates
func (h *LocationSSEHandler) StreamTechnicianEvents(e *pbCore.RequestEvent) error {
	techID := e.Request.PathValue("id")
	if techID == "" {
		return e.JSON(400, map[string]string{"error": "Missing technician ID"})
	}

	// Set SSE headers
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to tech channel
	eventChan := h.broker.Subscribe(broker.ChannelTech, techID)
	defer h.broker.Unsubscribe(broker.ChannelTech, techID, eventChan)

	// Send initial connection message
	sendSSEMessage(e, "connected", map[string]interface{}{
		"message": "Connected to technician events",
		"tech_id": techID,
	})

	// Heartbeat to keep connection alive every 30 seconds
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case event, ok := <-eventChan:
			if !ok {
				return nil
			}
			sendSSEMessage(e, event.Type, event.Data)

		case <-ticker.C:
			// Send heartbeat
			sendSSEMessage(e, "heartbeat", map[string]interface{}{
				"timestamp": time.Now().Unix(),
			})

		case <-e.Request.Context().Done():
			// Client disconnected
			log.Printf("📡 Client disconnected from tech %s", techID)
			return nil
		}
	}
}

// ============ HELPER FUNCTIONS ============

// sendSSEMessage sends a single SSE message
func sendSSEMessage(e *pbCore.RequestEvent, eventType string, data interface{}) {
	// Convert data to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("Failed to marshal event data: %v", err)
		return
	}

	// Format SSE message
	message := fmt.Sprintf("event: %s\n", eventType)
	message += fmt.Sprintf("data: %s\n\n", string(jsonData))

	// Write to response
	if _, err := e.Response.Write([]byte(message)); err != nil {
		log.Printf("Failed to write SSE message: %v", err)
		return
	}

	// Flush to ensure message is sent immediately
	if flusher, ok := e.Response.(interface{ Flush() }); ok {
		flusher.Flush()
	}
}

// sendSSEHeartbeat sends a heartbeat message to keep connection alive
func sendSSEHeartbeat(e *pbCore.RequestEvent) {
	if _, err := e.Response.Write([]byte(": heartbeat\n\n")); err != nil {
		log.Printf("Failed to write heartbeat: %v", err)
		return
	}
	if flusher, ok := e.Response.(interface{ Flush() }); ok {
		flusher.Flush()
	}
}
