// Package server maneja las conexiones WebSocket y el encolamiento de trabajos.
package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
	"nhooyr.io/websocket/wsjson"
)

// Config holds server configuration
type Config struct {
	QueueSize int
}

// PrintJob represents a queued print request
type PrintJob struct {
	ID         string          `json:"id"`
	ClientConn *websocket.Conn `json:"-"`
	Document   json.RawMessage `json:"data"`
	ReceivedAt time.Time       `json:"received_at"`
}

// Message represents incoming WebSocket message
type Message struct {
	Tipo  string          `json:"tipo"`
	ID    string          `json:"id,omitempty"`
	Datos json.RawMessage `json:"data,omitempty"`
}

// Response represents outgoing WebSocket message
type Response struct {
	Tipo     string `json:"tipo"`
	ID       string `json:"id,omitempty"`
	Status   string `json:"status,omitempty"`
	Mensaje  string `json:"mensaje,omitempty"`
	Position int    `json:"position,omitempty"`
}

// Server manages WebSocket connections and job queue
type Server struct {
	clients      *ClientRegistry
	jobQueue     chan *PrintJob
	queueSize    int
	shutdownOnce sync.Once
	shutdownChan chan struct{}
}

// NewServer creates a new WebSocket server
func NewServer(cfg Config) *Server {
	if cfg.QueueSize <= 0 {
		cfg.QueueSize = 100
	}

	return &Server{
		clients:      NewClientRegistry(),
		jobQueue:     make(chan *PrintJob, cfg.QueueSize),
		queueSize:    cfg.QueueSize,
		shutdownChan: make(chan struct{}),
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
		log.Printf("[X] Error al aceptar cliente: %v", err)
		return
	}

	// Register client
	s.clients.Add(conn)
	clientCount := s.clients.Count()
	log.Printf("[+] Cliente conectado (total: %d)", clientCount)

	// Send welcome message
	ctx := r.Context()
	welcome := Response{
		Tipo:    "info",
		Status:  "connected",
		Mensaje: "Conectado al servidor de impresión de tickets",
	}
	_ = wsjson.Write(ctx, conn, welcome)

	// Handle messages
	s.handleMessages(ctx, conn)

	// Cleanup on disconnect
	s.clients.Remove(conn)
	conn.Close(websocket.StatusNormalClosure, "desconectado")
	log.Printf("[-] Cliente desconectado (restantes: %d)", s.clients.Count())
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
			log.Printf("[!] Error leyendo mensaje: %v", err)
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
	default:
		s.sendError(ctx, conn, msg.ID, "Tipo de mensaje desconocido:  "+msg.Tipo)
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
		s.sendError(ctx, conn, jobID, "Campo 'datos' requerido para tipo 'ticket'")
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
		log.Printf("[>] Job encolado:  %s (cola: %d/%d)", jobID, current, capacity)

		response := Response{
			Tipo:     "ack",
			ID:       jobID,
			Status:   "queued",
			Position: current,
			Mensaje:  "Trabajo en cola",
		}
		_ = wsjson.Write(ctx, conn, response)

	default:
		// Queue full
		log.Printf("[!] Cola llena, rechazando job: %s", jobID)
		s.sendError(ctx, conn, jobID, "Cola llena, reintente en unos segundos")
	}
}

// handleStatus sends queue status
func (s *Server) handleStatus(ctx context.Context, conn *websocket.Conn) {
	current, capacity := s.QueueStatus()
	response := Response{
		Tipo:     "status",
		Status:   "ok",
		Position: current,
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

		// Notify all clients
		s.clients.ForEach(func(conn *websocket.Conn) {
			conn.Close(websocket.StatusGoingAway, "Servidor apagándose")
		})
	})
}

func formatStatus(current, capacity int) string {
	return "Cola: " + itoa(current) + "/" + itoa(capacity)
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune('0'+i%10)) + result
		i /= 10
	}
	return result
}
