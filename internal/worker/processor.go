// Package worker contiene la lógica del procesador de trabajos de impresión.
package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

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
	jobQueue      <-chan *server.PrintJob
	notifier      ClientNotifier
	config        Config
	stopChan      chan struct{}
	wg            sync.WaitGroup
	mu            sync.Mutex
	isRunning     bool
	jobsProcessed int64
	jobsFailed    int64
	lastJobTime   time.Time
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

	log.Println("[WORKER] ✅ Print worker started and ready")
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

	log.Printf("[WORKER] 🛑 Print worker stopped (processed:  %d, failed:  %d)", w.jobsProcessed, w.jobsFailed)
}

// run is the main worker loop
func (w *Worker) run() {
	defer w.wg.Done()

	log.Println("[WORKER] 👂 Waiting for print jobs...")

	for {
		select {
		case <-w.stopChan:
			log.Println("[WORKER] 📴 Received stop signal")
			return

		case job, ok := <-w.jobQueue:
			if !ok {
				log.Println("[WORKER] 📴 Job channel closed, exiting")
				return
			}
			w.processJob(job)
		}
	}
}

// processJob handles a single print job
func (w *Worker) processJob(job *server.PrintJob) {
	startTime := time.Now()
	log.Printf("[WORKER] 🔄 Processing job:  %s", job.ID)

	// Process the job
	err := w.executePrint(job)

	duration := time.Since(startTime)
	w.lastJobTime = time.Now()

	// Update statistics
	w.mu.Lock()
	if err != nil {
		w.jobsFailed++
	} else {
		w.jobsProcessed++
	}
	w.mu.Unlock()

	// Prepare response
	var response server.Response
	if err != nil {
		// Log detailed error to file for debugging
		log.Printf("[WORKER] ❌ Job %s FAILED after %v:  %v", job.ID, duration, err)

		// Extract user-friendly error message
		errorMsg := extractUserFriendlyError(err)

		response = server.Response{
			Tipo:    "result",
			ID:      job.ID,
			Status:  "error",
			Mensaje: errorMsg,
		}
	} else {
		log.Printf("[WORKER] ✅ Job %s completed in %v", job.ID, duration)
		response = server.Response{
			Tipo:    "result",
			ID:      job.ID,
			Status:  "success",
			Mensaje: fmt.Sprintf("Print completed in %v", duration.Round(time.Millisecond)),
		}
	}

	// Notify client (async to not block worker loop)
	if job.ClientConn != nil && w.notifier != nil {
		go func() {
			if err := w.notifier.NotifyClient(job.ClientConn, response); err != nil {
				log.Printf("[WORKER] ⚠️ Failed to notify client for job %s: %v", job.ID, err)
			}
		}()
	}
}

// extractUserFriendlyError creates a clean error message for the UI
func extractUserFriendlyError(err error) string {
	errStr := err.Error()

	// Common error patterns and their friendly messages
	errorMappings := []struct {
		pattern string
		message string
	}{
		{"version is required", "VALIDATION: Missing 'version' field"},
		{"profile.model is required", "VALIDATION: Missing 'profile.model' field"},
		{"at least one command", "VALIDATION: Document must contain at least one command"},
		{"invalid paper_width", "VALIDATION: Invalid paper width (use 58 or 80)"},
		{"invalid dpi", "VALIDATION: Invalid DPI value"},
		{"invalid version format", "VALIDATION: Invalid version format (use X.Y pattern)"},
		{"error conectando a impresora", "PRINTER: Cannot connect - check if printer is installed"},
		{"nombre de impresora no especificado", "PRINTER: No printer name specified in profile.model"},
		{"QR data cannot be empty", "QR:  Data cannot be empty"},
		{"QR data too long", "QR: Data exceeds maximum length"},
		{"barcode symbology is required", "BARCODE: Symbology type is required"},
		{"barcode data is required", "BARCODE: Data is required"},
		{"table overflow", "TABLE:  Columns exceed paper width"},
		{"raw command cannot be empty", "RAW: Command hex cannot be empty"},
		{"unsafe command blocked", "RAW:  Blocked by safe_mode - potentially dangerous command"},
		{"failed to load image", "IMAGE: Invalid or corrupted base64 data"},
		{"invalid QR correction level", "QR:  Invalid correction level (use L, M, Q, or H)"},
		{"unknown command type", "COMMAND:  Unknown command type"},
	}

	// Check for matching patterns
	for _, mapping := range errorMappings {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(mapping.pattern)) {
			return mapping.message
		}
	}

	// Categorize by error source
	if strings.Contains(errStr, "documento inválido") {
		return fmt.Sprintf("VALIDATION: %s", extractInnerError(errStr))
	}
	if strings.Contains(errStr, "error parseando") {
		return "JSON: Invalid document structure"
	}
	if strings.Contains(errStr, "error ejecutando") {
		return fmt.Sprintf("EXECUTION: %s", extractInnerError(errStr))
	}

	// Fallback:  return cleaned error
	return fmt.Sprintf("ERROR: %s", cleanErrorMessage(errStr))
}

// extractInnerError gets the innermost error message
func extractInnerError(errStr string) string {
	// Find the last colon-separated segment
	parts := strings.Split(errStr, ": ")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return errStr
}

// cleanErrorMessage removes verbose prefixes
func cleanErrorMessage(errStr string) string {
	// Remove common prefixes
	prefixes := []string{
		"error parseando documento:  ",
		"documento inválido: ",
		"error ejecutando documento: ",
		"error creando servicio de impresión: ",
	}
	result := errStr
	for _, prefix := range prefixes {
		result = strings.TrimPrefix(result, prefix)
	}
	return result
}

// executePrint performs the actual printing using Poster library
func (w *Worker) executePrint(job *server.PrintJob) (err error) {
	// Capturar panics y convertirlos en errores
	defer func() {
		if r := recover(); r != nil {
			// Asignar a la variable de retorno nombrada
			err = fmt.Errorf("panic recovered in executePrint: %v", r)
			log.Printf("[WORKER] 💥 Panic in job %s: %v\nStack:  %s",
				job.ID, r, debug.Stack())
		}
	}()

	// 1. Parse document from JSON
	var doc *schema.Document
	doc, err = w.parseDocument(job.Document)
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

	log.Printf("[WORKER] 🖨️ Job %s -> Printer: %s", job.ID, printerName)

	// 4. Create connection (open per job, close immediately after)
	conn, err := connection.NewWindowsPrintConnector(printerName)
	if err != nil {
		return fmt.Errorf("error conectando a impresora '%s': %w", printerName, err)
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Printf("[WORKER] ⚠️ Error closing connection: %v", err)
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
			log.Printf("[WORKER] ⚠️ Error closing printer: %v", err)
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
		IsRunning:     w.isRunning,
		JobsProcessed: w.jobsProcessed,
		JobsFailed:    w.jobsFailed,
		LastJobTime:   w.lastJobTime,
	}
}

// Statistics holds worker runtime statistics
type Statistics struct {
	IsRunning     bool      `json:"is_running"`
	JobsProcessed int64     `json:"jobs_processed"`
	JobsFailed    int64     `json:"jobs_failed"`
	LastJobTime   time.Time `json:"last_job_time,omitempty"`
}
