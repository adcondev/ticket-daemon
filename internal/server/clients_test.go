package server

import (
	"sync"
	"testing"

	"github.com/coder/websocket"
)

func TestClientRegistry_BasicOperations(t *testing.T) {
	registry := NewClientRegistry()

	// Create dummy connections
	// We use an array to make sure they have different addresses
	conns := make([]websocket.Conn, 3)
	conn1 := &conns[0]
	conn2 := &conns[1]
	conn3 := &conns[2]

	// Initial state
	if count := registry.Count(); count != 0 {
		t.Errorf("Expected 0 clients, got %d", count)
	}

	// Add clients
	registry.Add(conn1)
	registry.Add(conn2)

	if count := registry.Count(); count != 2 {
		t.Errorf("Expected 2 clients, got %d", count)
	}
	if !registry.Contains(conn1) {
		t.Error("Expected registry to contain conn1")
	}
	if registry.Contains(conn3) {
		t.Error("Expected registry to not contain conn3")
	}

	// Remove client
	registry.Remove(conn1)

	if count := registry.Count(); count != 1 {
		t.Errorf("Expected 1 client, got %d", count)
	}
	if registry.Contains(conn1) {
		t.Error("Expected registry to not contain conn1 after removal")
	}
	if !registry.Contains(conn2) {
		t.Error("Expected registry to contain conn2")
	}
}

func TestClientRegistry_ForEachAndBroadcast(t *testing.T) {
	registry := NewClientRegistry()

	conns := make([]websocket.Conn, 2)
	conn1 := &conns[0]
	conn2 := &conns[1]

	registry.Add(conn1)
	registry.Add(conn2)

	// Test ForEach
	visited := make(map[*websocket.Conn]bool)
	registry.ForEach(func(c *websocket.Conn) {
		visited[c] = true
	})

	if len(visited) != 2 || !visited[conn1] || !visited[conn2] {
		t.Errorf("ForEach did not visit all clients correctly: %v", visited)
	}

	// Test Broadcast
	visitedBroadcast := make(map[*websocket.Conn]bool)
	registry.Broadcast(func(c *websocket.Conn) error {
		visitedBroadcast[c] = true
		return nil
	})

	if len(visitedBroadcast) != 2 || !visitedBroadcast[conn1] || !visitedBroadcast[conn2] {
		t.Errorf("Broadcast did not visit all clients correctly: %v", visitedBroadcast)
	}
}

func TestClientRegistry_Concurrency(t *testing.T) {
	registry := NewClientRegistry()
	var wg sync.WaitGroup

	numGoroutines := 100
	conns := make([]websocket.Conn, numGoroutines)

	// Concurrent Add
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			registry.Add(&conns[i])
		}(i)
	}
	wg.Wait()

	if count := registry.Count(); count != numGoroutines {
		t.Errorf("Expected %d clients, got %d", numGoroutines, count)
	}

	// Concurrent Contains, ForEach, Broadcast
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = registry.Contains(&conns[i])
			if i%10 == 0 {
				registry.ForEach(func(_ *websocket.Conn) {})
				registry.Broadcast(func(_ *websocket.Conn) error { return nil })
			}
		}(i)
	}
	wg.Wait()

	// Concurrent Remove
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			registry.Remove(&conns[i])
		}(i)
	}
	wg.Wait()

	if count := registry.Count(); count != 0 {
		t.Errorf("Expected 0 clients after removal, got %d", count)
	}
}
