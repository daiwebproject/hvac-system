package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"hvac-system/pkg/broker"

	"github.com/pocketbase/pocketbase/core"
)

// CustomerTrackStream provides SSE endpoint for customer order tracking
// Customer receives only events for their specific booking
func (h *WebHandler) CustomerTrackStream(e *core.RequestEvent) error {
	bookingID := e.Request.PathValue("booking_id")
	if bookingID == "" {
		return e.String(400, "Missing booking_id")
	}

	// TODO: Verify tracking token in query params
	// trackingToken := e.Request.URL.Query().Get("token")
	// For now, allow public access (can add token verification later)

	// Verify booking exists
	_, err := h.App.FindRecordById("bookings", bookingID)
	if err != nil {
		return e.String(404, "Booking not found")
	}

	// Set SSE headers
	e.Response.Header().Set("Content-Type", "text/event-stream")
	e.Response.Header().Set("Cache-Control", "no-cache")
	e.Response.Header().Set("Connection", "keep-alive")
	e.Response.Header().Set("Access-Control-Allow-Origin", "*") // Allow cross-origin for tracking page

	// Subscribe to customer-specific channel
	eventChan := h.Broker.Subscribe(broker.ChannelCustomer, bookingID)
	defer h.Broker.Unsubscribe(broker.ChannelCustomer, bookingID, eventChan)

	// Send initial connection event
	initialEvent := broker.Event{
		Type:      "connection.established",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"role":       "customer",
			"booking_id": bookingID,
		},
	}

	eventJSON, _ := json.Marshal(initialEvent)
	fmt.Fprintf(e.Response, "data: %s\n\n", eventJSON)
	e.Response.(http.Flusher).Flush()

	// Stream events (only for this booking)
	for {
		select {
		case event := <-eventChan:
			eventJSON, err := json.Marshal(event)
			if err != nil {
				continue
			}
			fmt.Fprintf(e.Response, "data: %s\n\n", eventJSON)
			e.Response.(http.Flusher).Flush()

		case <-e.Request.Context().Done():
			// Client disconnected
			return nil
		}
	}
}
