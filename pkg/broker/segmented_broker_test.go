package broker

import (
	"testing"
	"time"
)

func TestSegmentedBroker_AdminChannel(t *testing.T) {
	broker := NewSegmentedBroker()

	// Subscribe admin clients
	client1 := broker.Subscribe(ChannelAdmin, "")
	client2 := broker.Subscribe(ChannelAdmin, "")

	// Publish event
	event := Event{
		Type:      "booking.created",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"booking_id": "123",
		},
	}

	go broker.Publish(ChannelAdmin, "", event)

	// Both clients should receive
	select {
	case e := <-client1:
		if e.Type != "booking.created" {
			t.Errorf("Expected booking.created, got %s", e.Type)
		}
	case <-time.After(time.Second):
		t.Error("Client 1 timeout")
	}

	select {
	case e := <-client2:
		if e.Type != "booking.created" {
			t.Errorf("Expected booking.created, got %s", e.Type)
		}
	case <-time.After(time.Second):
		t.Error("Client 2 timeout")
	}
}

func TestSegmentedBroker_TechChannel_Isolation(t *testing.T) {
	broker := NewSegmentedBroker()

	// Subscribe two different techs
	techA := broker.Subscribe(ChannelTech, "tech_a")
	techB := broker.Subscribe(ChannelTech, "tech_b")

	// Publish to tech A only
	event := Event{
		Type:      "job.assigned",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"tech_id": "tech_a",
		},
	}

	go broker.Publish(ChannelTech, "tech_a", event)

	// Tech A should receive
	select {
	case e := <-techA:
		if e.Type != "job.assigned" {
			t.Errorf("Expected job.assigned, got %s", e.Type)
		}
	case <-time.After(time.Second):
		t.Error("Tech A timeout")
	}

	// Tech B should NOT receive
	select {
	case <-techB:
		t.Error("Tech B should not receive event meant for Tech A")
	case <-time.After(100 * time.Millisecond):
		// Expected: timeout means no event received
	}
}

func TestSegmentedBroker_CustomerChannel(t *testing.T) {
	broker := NewSegmentedBroker()

	// Subscribe to specific booking
	customer := broker.Subscribe(ChannelCustomer, "booking_123")

	// Publish location update
	event := Event{
		Type:      "tech.location",
		Timestamp: time.Now().Unix(),
		Data: map[string]interface{}{
			"booking_id": "booking_123",
			"lat":        10.762622,
			"lng":        106.660172,
		},
	}

	go broker.Publish(ChannelCustomer, "booking_123", event)

	// Customer should receive
	select {
	case e := <-customer:
		if e.Type != "tech.location" {
			t.Errorf("Expected tech.location, got %s", e.Type)
		}
	case <-time.After(time.Second):
		t.Error("Customer timeout")
	}
}

func TestSegmentedBroker_Unsubscribe(t *testing.T) {
	broker := NewSegmentedBroker()

	client := broker.Subscribe(ChannelAdmin, "")

	// Check stats before unsubscribe
	stats := broker.GetStats()
	if stats["admin_clients"] != 1 {
		t.Errorf("Expected 1 admin client, got %d", stats["admin_clients"])
	}

	// Unsubscribe
	broker.Unsubscribe(ChannelAdmin, "", client)

	// Check stats after unsubscribe
	stats = broker.GetStats()
	if stats["admin_clients"] != 0 {
		t.Errorf("Expected 0 admin clients, got %d", stats["admin_clients"])
	}
}

func TestSegmentedBroker_GetStats(t *testing.T) {
	broker := NewSegmentedBroker()

	broker.Subscribe(ChannelAdmin, "")
	broker.Subscribe(ChannelAdmin, "")
	broker.Subscribe(ChannelTech, "tech_1")
	broker.Subscribe(ChannelTech, "tech_2")
	broker.Subscribe(ChannelCustomer, "booking_1")

	stats := broker.GetStats()

	if stats["admin_clients"] != 2 {
		t.Errorf("Expected 2 admin clients, got %d", stats["admin_clients"])
	}

	if stats["tech_clients"] != 2 {
		t.Errorf("Expected 2 tech clients, got %d", stats["tech_clients"])
	}

	if stats["customer_clients"] != 1 {
		t.Errorf("Expected 1 customer client, got %d", stats["customer_clients"])
	}
}
