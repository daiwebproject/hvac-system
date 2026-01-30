package broker

import (
	"sync"
)

// Channel represents the type of event channel
type Channel string

const (
	ChannelAdmin    Channel = "admin"
	ChannelTech     Channel = "tech"
	ChannelCustomer Channel = "customer"
)

// Event represents a system event
type Event struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// SegmentedBroker manages channel-based event distribution
type SegmentedBroker struct {
	// Admin channel: all clients receive all events
	adminClients map[chan Event]bool

	// Tech channels: map[tech_id]map[client_channel]bool
	// Each tech only receives events for their jobs
	techClients map[string]map[chan Event]bool

	// Customer channels: map[booking_id]map[client_channel]bool
	// Each customer only receives events for their booking
	customerClients map[string]map[chan Event]bool

	mutex sync.RWMutex
}

// NewSegmentedBroker creates a new segmented broker instance
func NewSegmentedBroker() *SegmentedBroker {
	return &SegmentedBroker{
		adminClients:    make(map[chan Event]bool),
		techClients:     make(map[string]map[chan Event]bool),
		customerClients: make(map[string]map[chan Event]bool),
	}
}

// Subscribe creates a new client channel and returns it
// For admin: id is ignored
// For tech: id is tech_id
// For customer: id is booking_id
func (b *SegmentedBroker) Subscribe(channel Channel, id string) chan Event {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	clientChan := make(chan Event, 10) // Buffered to prevent blocking

	switch channel {
	case ChannelAdmin:
		b.adminClients[clientChan] = true

	case ChannelTech:
		if _, exists := b.techClients[id]; !exists {
			b.techClients[id] = make(map[chan Event]bool)
		}
		b.techClients[id][clientChan] = true

	case ChannelCustomer:
		if _, exists := b.customerClients[id]; !exists {
			b.customerClients[id] = make(map[chan Event]bool)
		}
		b.customerClients[id][clientChan] = true
	}

	return clientChan
}

// Unsubscribe removes a client channel
func (b *SegmentedBroker) Unsubscribe(channel Channel, id string, clientChan chan Event) {
	b.mutex.Lock()
	defer b.mutex.Unlock()

	switch channel {
	case ChannelAdmin:
		delete(b.adminClients, clientChan)
		close(clientChan)

	case ChannelTech:
		if clients, exists := b.techClients[id]; exists {
			delete(clients, clientChan)
			if len(clients) == 0 {
				delete(b.techClients, id)
			}
		}
		close(clientChan)

	case ChannelCustomer:
		if clients, exists := b.customerClients[id]; exists {
			delete(clients, clientChan)
			if len(clients) == 0 {
				delete(b.customerClients, id)
			}
		}
		close(clientChan)
	}
}

// Publish sends an event to the appropriate channel(s)
// For admin: publishes to all admin clients
// For tech: publishes to specific tech's clients
// For customer: publishes to specific booking's clients
func (b *SegmentedBroker) Publish(channel Channel, id string, event Event) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	switch channel {
	case ChannelAdmin:
		// Send to all admin clients
		for clientChan := range b.adminClients {
			select {
			case clientChan <- event:
			default:
				// Client not ready, skip to avoid blocking
			}
		}

	case ChannelTech:
		// Send only to specific tech's clients
		if clients, exists := b.techClients[id]; exists {
			for clientChan := range clients {
				select {
				case clientChan <- event:
				default:
					// Client not ready, skip
				}
			}
		}

	case ChannelCustomer:
		// Send only to specific booking's clients
		if clients, exists := b.customerClients[id]; exists {
			for clientChan := range clients {
				select {
				case clientChan <- event:
				default:
					// Client not ready, skip
				}
			}
		}
	}
}

// PublishToAll sends event to all channels (for critical system events)
func (b *SegmentedBroker) PublishToAll(event Event) {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	// Send to all admin clients
	for clientChan := range b.adminClients {
		select {
		case clientChan <- event:
		default:
		}
	}

	// Send to all tech clients
	for _, clients := range b.techClients {
		for clientChan := range clients {
			select {
			case clientChan <- event:
			default:
			}
		}
	}

	// Send to all customer clients
	for _, clients := range b.customerClients {
		for clientChan := range clients {
			select {
			case clientChan <- event:
			default:
			}
		}
	}
}

// GetStats returns current broker statistics
func (b *SegmentedBroker) GetStats() map[string]int {
	b.mutex.RLock()
	defer b.mutex.RUnlock()

	techCount := 0
	for _, clients := range b.techClients {
		techCount += len(clients)
	}

	customerCount := 0
	for _, clients := range b.customerClients {
		customerCount += len(clients)
	}

	return map[string]int{
		"admin_clients":    len(b.adminClients),
		"tech_clients":     techCount,
		"customer_clients": customerCount,
	}
}
