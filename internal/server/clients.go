// Package server maneja las conexiones WebSocket y el encolamiento de trabajos.
package server

import (
	"sync"

	"github.com/coder/websocket"
)

// ClientRegistry manages connected WebSocket clients thread-safely
type ClientRegistry struct {
	clients map[*websocket.Conn]*sync.Mutex
	mu      sync.RWMutex
}

// NewClientRegistry creates a new client registry
func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{
		clients: make(map[*websocket.Conn]*sync.Mutex),
	}
}

// Add registers a new client connection
func (r *ClientRegistry) Add(conn *websocket.Conn) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[conn] = &sync.Mutex{}
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
	_, ok := r.clients[conn]
	return ok
}

// GetMutex returns the mutex for a given connection, if it exists
func (r *ClientRegistry) GetMutex(conn *websocket.Conn) *sync.Mutex {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.clients[conn]
}

// ForEach executes a function for each connected client
func (r *ClientRegistry) ForEach(fn func(*websocket.Conn)) {
	r.mu.RLock()
	// Copy keys to avoid holding lock during iteration
	conns := make([]*websocket.Conn, 0, len(r.clients))
	for conn := range r.clients {
		conns = append(conns, conn)
	}
	r.mu.RUnlock()

	for _, conn := range conns {
		fn(conn)
	}
}

// Broadcast sends a message to all connected clients
func (r *ClientRegistry) Broadcast(fn func(*websocket.Conn) error) {
	r.mu.RLock()
	conns := make([]*websocket.Conn, 0, len(r.clients))
	for conn := range r.clients {
		conns = append(conns, conn)
	}
	r.mu.RUnlock()

	for _, conn := range conns {
		_ = fn(conn)
	}
}
