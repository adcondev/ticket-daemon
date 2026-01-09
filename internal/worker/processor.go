// Package worker contiene la lógica del procesador de trabajos de impresión.
package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/adcondev/poster/pkg/composer"
	"github.com/adcondev/poster/pkg/connection"
	"github.com/adcondev/poster/pkg/document/executor"
	"github.com/adcondev/poster/pkg/document/schema"
	"github.com/adcondev/poster/pkg/profile"
	"github.com/adcondev/poster/pkg/service"
	"github.com/adcondev/ticket-daemon/internal/server"
)

// Config holds worker configuration
type Config struct {
	DefaultPrinter string // Fallback printer name if not specified in document
}

// ClientNotifier interface for sending results back to clients
type ClientNotifier interface {
	NotifyClient(conn *websocket.Conn, response server.Response) error
}

// Worker consumes print jobs from the queue and executes them via Poster
type Worker struct {
	jobQueue  <-chan *server.PrintJob
	notifier  ClientNotifier
	config    Config
	stopChan  chan struct{}
	wg        sync.WaitGroup
	mu        sync.Mutex // Serialize printer operations (safety)
	isRunning bool
}

// NewWorker creates a new print worker
func NewWorker(jobQueue <-chan *server.PrintJob, notifier ClientNotifier, config Config) *Worker {
	return &Worker{
		jobQueue: jobQueue,
		notifier: notifier,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start begins the worker goroutine
func (w *Worker) Start() {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = true
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	log.Println("[OK] Worker de impresión iniciado")
}

// Stop gracefully stops the worker
func (w *Worker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = false
	w.mu.Unlock()

	close(w.stopChan)
	w.wg.Wait()

	log.Println("[OK] Worker de impresión detenido")
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.wg.Done()

	log.Println("[i] Worker esperando trabajos de impresión...")

	for {
		select {
		case <-w.stopChan:
			log.Println("[i] Worker recibió señal de parada")
			return

		case job, ok := <-w.jobQueue:
			if !ok {
				log.Println("[i] Canal de trabajos cerrado, terminando worker")
				return
			}
			w.processJob(job)
		}
	}
}

// processJob handles a single print job
func (w *Worker) processJob(job *server.PrintJob) {
	startTime := time.Now()
	log.Printf("[>] Procesando job: %s", job.ID)

	// Process the job
	err := w.executePrint(job)

	duration := time.Since(startTime)

	// Prepare response
	var response server.Response
	if err != nil {
		log.Printf("[X] Error en job %s: %v (duración: %v)", job.ID, err, duration)
		response = server.Response{
			Tipo:    "result",
			ID:      job.ID,
			Status:  "error",
			Mensaje: fmt.Sprintf("Error de impresión: %v", err),
		}
	} else {
		log.Printf("[OK] Job %s completado exitosamente (duración: %v)", job.ID, duration)
		response = server.Response{
			Tipo:    "result",
			ID:      job.ID,
			Status:  "success",
			Mensaje: "Impresión completada",
		}
	}

	// Notify client
	if job.ClientConn != nil && w.notifier != nil {
		if err := w.notifier.NotifyClient(job.ClientConn, response); err != nil {
			log.Printf("[! ] Error notificando cliente para job %s: %v", job.ID, err)
		}
	}
}

// executePrint performs the actual printing using Poster library
func (w *Worker) executePrint(job *server.PrintJob) error {
	// 1. Parse document from JSON
	doc, err := w.parseDocument(job.Document)
	if err != nil {
		return fmt.Errorf("error parseando documento: %w", err)
	}

	// 2. Validate document
	if err := doc.Validate(); err != nil {
		return fmt.Errorf("documento inválido: %w", err)
	}

	// 3. Get printer name
	printerName := w.getPrinterName(doc)
	if printerName == "" {
		return fmt.Errorf("nombre de impresora no especificado")
	}

	log.Printf("[i] Job %s -> Impresora: %s", job.ID, printerName)

	// 4. Create connection (open per job, close immediately after)
	conn, err := connection.NewWindowsPrintConnector(printerName)
	if err != nil {
		return fmt.Errorf("error conectando a impresora '%s': %w", printerName, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("[!] Error cerrando conexión:  %v", err)
		}
	}()

	// 5. Create profile based on document
	prof := w.createProfile(doc)

	// 6. Create protocol (ESC/POS)
	proto := composer.NewEscpos()

	// 7. Create printer service
	printer, err := service.NewPrinter(proto, prof, conn)
	if err != nil {
		return fmt.Errorf("error creando servicio de impresión: %w", err)
	}
	defer func() {
		if err := printer.Close(); err != nil {
			log.Printf("[! ] Error cerrando printer: %v", err)
		}
	}()

	// 8. Create executor and print
	exec := executor.NewExecutor(printer)

	if err := exec.Execute(doc); err != nil {
		return fmt.Errorf("error ejecutando documento: %w", err)
	}

	return nil
}

// parseDocument converts raw JSON to schema.Document
func (w *Worker) parseDocument(rawJSON json.RawMessage) (*schema.Document, error) {
	var doc schema.Document

	if err := json.Unmarshal(rawJSON, &doc); err != nil {
		return nil, fmt.Errorf("JSON inválido: %w", err)
	}

	return &doc, nil
}

// getPrinterName extracts printer name from document or uses default
func (w *Worker) getPrinterName(doc *schema.Document) string {
	// Priority 1: Document profile model
	if doc.Profile.Model != "" {
		return doc.Profile.Model
	}

	// Priority 2: Worker default config
	if w.config.DefaultPrinter != "" {
		return w.config.DefaultPrinter
	}

	return ""
}

// createProfile creates a Poster profile based on document configuration
func (w *Worker) createProfile(doc *schema.Document) *profile.Escpos {
	var prof *profile.Escpos

	// Select profile based on paper width or model
	if doc.Profile.Model != "" {
		switch strings.ToLower(doc.Profile.Model) {
		case "80mm ec-pm-80250", "ec-pm-80250":
			prof = profile.CreateECPM80250()
		case "58mm pt-210", "pt-210":
			prof = profile.CreatePt210()
		case "58mm gp-58n", "gp-58n":
			prof = profile.CreateGP58N()
		default:
			// Default based on paper width
			if doc.Profile.PaperWidth >= 80 {
				prof = profile.CreateProfile80mm()
			} else {
				prof = profile.CreateProfile58mm()
			}
		}
	} else {
		// Default based on paper width
		if doc.Profile.PaperWidth >= 80 {
			prof = profile.CreateProfile80mm()
		} else {
			prof = profile.CreateProfile58mm()
		}
	}

	// Apply JSON overrides
	if doc.Profile.Model != "" {
		prof.Model = doc.Profile.Model
	}
	if doc.Profile.DPI > 0 {
		prof.DPI = doc.Profile.DPI
	}
	prof.HasQR = doc.Profile.HasQR

	return prof
}

// Stats returns current worker statistics
func (w *Worker) Stats() Statistics {
	w.mu.Lock()
	defer w.mu.Unlock()

	return Statistics{
		IsRunning: w.isRunning,
	}
}

// Statistics holds worker runtime statistics
type Statistics struct {
	IsRunning bool `json:"is_running"`
}
