// Package server maneja las conexiones WebSocket y el encolamiento de trabajos.
package server

import (
	"sync"

	"nhooyr.io/websocket"
)

// ClientRegistry manages connected WebSocket clients thread-safely
type ClientRegistry struct {
	clients map[*websocket.Conn]bool
	mu      sync.RWMutex
}

// NewClientRegistry creates a new client registry
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[*websocket.Conn]bool),
	}
}

// Add registers a new client connection
func (r *ClientRegistry) Add(conn *websocket.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[conn] = true
}

// Remove unregisters a client connection
func (r *ClientRegistry) Remove(conn *websocket.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.clients, conn)
}

// Count returns the number of connected clients
func (r *ClientRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.clients)
}

// Contains checks if a client is registered
func (r *ClientRegistry) Contains(conn *websocket.Conn) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[conn]
}

// ForEach executes a function for each connected client
func (r *ClientRegistry) ForEach(fn func(*websocket.Conn)) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for conn := range r.clients {
		fn(conn)
	}
}

// Broadcast sends a message to all connected clients
func (r *ClientRegistry) Broadcast(fn func(*websocket.Conn) error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for conn := range r.clients {
		_ = fn(conn)
	}
}
