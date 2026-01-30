package realtime

import (
	"fmt"
	"sync"
)

// Broker quản lý các kết nối SSE
type Broker struct {
	clients    map[chan string]bool
	register   chan chan string
	unregister chan chan string
	broadcast  chan string
	mutex      sync.Mutex
}

// NewBroker khởi tạo
func NewBroker() *Broker {
	return &Broker{
		clients:    make(map[chan string]bool),
		register:   make(chan chan string),
		unregister: make(chan chan string),
		broadcast:  make(chan string),
	}
}

// Run lắng nghe các sự kiện
func (b *Broker) Run() {
	for {
		select {
		case client := <-b.register:
			b.mutex.Lock()
			b.clients[client] = true
			b.mutex.Unlock()
			fmt.Println("Client connected")
		case client := <-b.unregister:
			b.mutex.Lock()
			if _, ok := b.clients[client]; ok {
				delete(b.clients, client)
				close(client)
			}
			b.mutex.Unlock()
			fmt.Println("Client disconnected")
		case message := <-b.broadcast:
			b.mutex.Lock()
			for client := range b.clients {
				select {
				case client <- message:
				default:
					// Bỏ qua nếu client bị block
				}
			}
			b.mutex.Unlock()
		}
	}
}

// Send gửi tin nhắn tới tất cả client
func (b *Broker) Send(msg string) {
	b.broadcast <- msg
}

// AddClient thêm client mới
func (b *Broker) AddClient() chan string {
	client := make(chan string, 1)
	b.register <- client
	return client
}

// RemoveClient xóa client
func (b *Broker) RemoveClient(client chan string) {
	b.unregister <- client
}