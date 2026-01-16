// Package server maneja las conexiones WebSocket y el encolamiento de trabajos.
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"

	"github.com/adcondev/poster/pkg/connection"
	"github.com/adcondev/ticket-daemon/internal/printer"
)

type PrinterLister interface {
	GetPrinters(forceRefresh bool) ([]connection.PrinterDetail, error)
	GetSummary() printer.Summary
}

// Config holds server configuration
type Config struct {
	QueueSize int
}

// PrintJob represents a queued print request
type PrintJob struct {
	ID         string          `json:"id"`
	ClientConn *websocket.Conn `json:"-"`
	Document   json.RawMessage `json:"datos"`
	ReceivedAt time.Time       `json:"received_at"`
}

// Message represents incoming WebSocket message
type Message struct {
	Tipo  string          `json:"tipo"`
	ID    string          `json:"id,omitempty"`
	Datos json.RawMessage `json:"datos,omitempty"`
}

// Response represents outgoing WebSocket message
type Response struct {
	Tipo     string `json:"tipo"`
	ID       string `json:"id,omitempty"`
	Status   string `json:"status,omitempty"`
	Mensaje  string `json:"mensaje,omitempty"`
	Current  int    `json:"current,omitempty"`
	Capacity int    `json:"capacity,omitempty"`
}

// Server manages WebSocket connections and job queue
type Server struct {
	clients          *ClientRegistry
	jobQueue         chan *PrintJob
	queueSize        int
	shutdownOnce     sync.Once
	shutdownChan     chan struct{}
	printerDiscovery PrinterLister
}

// NewServer creates a new WebSocket server
func NewServer(cfg Config, discovery PrinterLister) *Server {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}

	return &Server{
		clients:          NewClientRegistry(),
		jobQueue:         make(chan *PrintJob, cfg.QueueSize),
		queueSize:        cfg.QueueSize,
		shutdownChan:     make(chan struct{}),
		printerDiscovery: discovery,
	}
}

// QueueStatus returns current and max queue size
func (s *Server) QueueStatus() (current, capacity int) {
	return len(s.jobQueue), cap(s.jobQueue)
}

// JobQueue returns the job queue channel (for worker consumption)
func (s *Server) JobQueue() <-chan *PrintJob {
	return s.jobQueue
}

// HandleWebSocket handles WebSocket connections
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Printf("[WS] ❌ Error accepting client: %v", err)
		return
	}

	// Register client
	s.clients.Add(conn)
	clientCount := s.clients.Count()
	log.Printf("[WS] ➕ Client connected (total: %d) from %s", clientCount, r.RemoteAddr)

	// Send welcome message
	ctx := r.Context()
	welcome := Response{
		Tipo:    "info",
		Status:  "connected",
		Mensaje: "✅ Servidor respondiendo desde Ticket Daemon",
	}
	_ = wsjson.Write(ctx, conn, welcome)

	// Handle messages
	s.handleMessages(ctx, conn)

	// Cleanup on disconnect
	s.clients.Remove(conn)
	err = conn.Close(websocket.StatusNormalClosure, "disconnected")
	if err != nil {
		return
	}
	log.Printf("[WS] ➖ Client disconnected (remaining: %d)", s.clients.Count())
}

// handleMessages processes incoming messages from a client
func (s *Server) handleMessages(ctx context.Context, conn *websocket.Conn) {
	for {
		select {
		case <-s.shutdownChan:
			return
		default:
		}

		var msg Message
		err := wsjson.Read(ctx, conn, &msg)
		if err != nil {
			// Normal closure or context cancelled
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				ctx.Err() != nil {
				return
			}
			log.Printf("[WS] ⚠️ Error reading message: %v", err)
			return
		}

		s.routeMessage(ctx, conn, &msg)
	}
}

// routeMessage routes message to appropriate handler
func (s *Server) routeMessage(ctx context.Context, conn *websocket.Conn, msg *Message) {
	switch msg.Tipo {
	case "ticket":
		s.handleTicket(ctx, conn, msg)
	case "status":
		s.handleStatus(ctx, conn)
	case "ping":
		s.handlePing(ctx, conn, msg)
	case "get_printers": // NEW
		s.handleGetPrinters(ctx, conn)
	default:
		log.Printf("[WS] ⚠️ Unknown message type: %s", msg.Tipo)
		s.sendError(ctx, conn, msg.ID, "Unknown message type:  "+msg.Tipo)
	}
}

// handleTicket processes a print job request
func (s *Server) handleTicket(ctx context.Context, conn *websocket.Conn, msg *Message) {
	// Generate ID if not provided
	jobID := msg.ID
	if jobID == "" {
		jobID = uuid.New().String()
	}

	// Validate document exists
	if len(msg.Datos) == 0 {
		log.Printf("[QUEUE] ❌ Job %s rejected: missing 'datos' field", jobID)
		s.sendError(ctx, conn, jobID, "Field 'datos' is required for type 'ticket'")
		return
	}

	// Create job
	job := &PrintJob{
		ID:         jobID,
		ClientConn: conn,
		Document:   msg.Datos,
		ReceivedAt: time.Now(),
	}

	// Try to enqueue (non-blocking)
	select {
	case s.jobQueue <- job:
		current, capacity := s.QueueStatus()
		log.Printf("[QUEUE] 📥 Job queued: %s (queue: %d/%d)", jobID, current, capacity)

		response := Response{
			Tipo:     "ack",
			ID:       jobID,
			Status:   "queued",
			Current:  current,
			Capacity: capacity,
			Mensaje:  "Job queued for printing",
		}

		log.Printf("[DEBUG] Sending ACK:  current=%d, capacity=%d", current, capacity)

		_ = wsjson.Write(ctx, conn, response)

	default:
		// Queue full
		current, capacity := s.QueueStatus()
		log.Printf("[QUEUE] 🚫 Queue full, rejecting job: %s (%d/%d)", jobID, current, capacity)
		s.sendError(ctx, conn, jobID, "Queue full, please retry in a few seconds")
	}
}

// handleStatus sends queue status
func (s *Server) handleStatus(ctx context.Context, conn *websocket.Conn) {
	current, capacity := s.QueueStatus()

	response := Response{
		Tipo:     "status",
		Status:   "ok",
		Current:  current,
		Capacity: capacity,
		Mensaje:  formatStatus(current, capacity),
	}
	_ = wsjson.Write(ctx, conn, response)
}

// handlePing responds to ping
func (s *Server) handlePing(ctx context.Context, conn *websocket.Conn, msg *Message) {
	response := Response{
		Tipo:   "pong",
		ID:     msg.ID,
		Status: "ok",
	}
	_ = wsjson.Write(ctx, conn, response)
}

// handleGetPrinters handles printer enumeration requests
func (s *Server) handleGetPrinters(ctx context.Context, conn *websocket.Conn) {
	printers, err := s.printerDiscovery.GetPrinters(false)
	if err != nil {
		s.sendError(ctx, conn, "", "Failed to enumerate printers: "+err.Error())
		return
	}

	// Convert to DTOs
	dtos := make([]printer.DetailDTO, len(printers))
	for i, p := range printers {
		dtos[i] = printer.DetailDTO{
			Name:        p.Name,
			Port:        p.Port,
			Driver:      p.Driver,
			Status:      string(p.Status),
			IsDefault:   p.IsDefault,
			IsVirtual:   p.IsVirtual,
			PrinterType: p.PrinterType,
		}
	}

	response := struct {
		Tipo     string              `json:"tipo"`
		Status   string              `json:"status"`
		Printers []printer.DetailDTO `json:"printers"`
		Summary  printer.Summary     `json:"summary"`
	}{
		Tipo:     "printers",
		Status:   "ok",
		Printers: dtos,
		Summary:  s.printerDiscovery.GetSummary(),
	}

	_ = wsjson.Write(ctx, conn, response)
}

// sendError sends error response to client
func (s *Server) sendError(ctx context.Context, conn *websocket.Conn, id, mensaje string) {
	response := Response{
		Tipo:    "error",
		ID:      id,
		Status:  "error",
		Mensaje: mensaje,
	}
	_ = wsjson.Write(ctx, conn, response)
}

// NotifyClient sends a result back to a specific client
func (s *Server) NotifyClient(conn *websocket.Conn, response Response) error {
	if conn == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return wsjson.Write(ctx, conn, response)
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown() {
	s.shutdownOnce.Do(func() {
		close(s.shutdownChan)

		clientCount := s.clients.Count()
		log.Printf("[WS] 🛑 Shutting down, disconnecting %d clients", clientCount)

		// Notify all clients
		s.clients.ForEach(func(conn *websocket.Conn) {
			err := conn.Close(websocket.StatusGoingAway, "Server shutting down")
			if err != nil {
				return
			}
		})
	})
}

func formatStatus(current, capacity int) string {
	return "Queue: " + strconv.Itoa(current) + "/" + strconv.Itoa(capacity)
}
