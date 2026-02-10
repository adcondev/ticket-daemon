package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/judwhite/go-svc"

	embed "github.com/adcondev/ticket-daemon"
	"github.com/adcondev/ticket-daemon/internal/config"
	"github.com/adcondev/ticket-daemon/internal/server"
	"github.com/adcondev/ticket-daemon/internal/worker"
)

// Build variables, injected at compile time
var (
	BuildEnvironment = "local"
	BuildDate        = "unknown"
	BuildTime        = "unknown"
)

// GetEnvConfig returns the current environment configuration
func GetEnvConfig() config.Environment {
	return config.GetEnvironment(BuildEnvironment)
}

// Program implements svc.Service interface
type Program struct {
	wg               sync.WaitGroup
	quit             chan struct{}
	httpServer       *http.Server
	wsServer         *server.Server
	printWorker      *worker.Worker
	startTime        time.Time
	printerDiscovery *PrinterDiscovery
}

// Init initializes the service
func (p *Program) Init(env svc.Environment) error {
	envConfig := GetEnvConfig()

	if err := initLogging(envConfig); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║   🎫 TICKET DAEMON - POS Print Service                     ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Printf("[INIT] 🚀 Starting service - Environment: %s", envConfig.Name)
	log.Printf("[INIT] 📅 Build: %s %s", BuildDate, BuildTime)

	return nil
}

// Start starts the service
func (p *Program) Start() error {
	p.quit = make(chan struct{})
	p.startTime = time.Now()
	cfg := GetEnvConfig()

	p.printerDiscovery = NewPrinterDiscovery()
	p.printerDiscovery.LogStartupDiagnostics()

	// Initialize WebSocket server
	p.wsServer = server.NewServer(server.Config{QueueSize: cfg.QueueCapacity}, p.printerDiscovery)

	// Initialize print worker
	p.printWorker = worker.NewWorker(
		p.wsServer.JobQueue(),
		p.wsServer,
		worker.Config{
			DefaultPrinter: cfg.DefaultPrinter,
		},
	)
	p.printWorker.Start()

	// Create HTTP server
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", p.wsServer.HandleWebSocket)

	// Enhanced health check endpoint with more metrics
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		current, capacity := p.wsServer.QueueStatus()
		stats := p.printWorker.Stats()

		response := HealthResponse{
			Status: "ok",
			Queue: QueueStatus{
				Current:     current,
				Capacity:    capacity,
				Utilization: float64(current) / float64(capacity) * 100,
			},
			Worker: WorkerStatus{
				Running:       stats.IsRunning,
				JobsProcessed: stats.JobsProcessed,
				JobsFailed:    stats.JobsFailed,
			},
			Printers: p.printerDiscovery.GetSummary(), // NEW
			Build: BuildInfo{
				Env:  BuildEnvironment,
				Date: BuildDate,
				Time: BuildTime,
			},
			Uptime: int(time.Since(p.startTime).Seconds()),
		}

		// NEW: Adjust overall status based on printer subsystem
		if response.Printers.Status == "error" {
			response.Status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_ = json.NewEncoder(w).Encode(response)
	})

	// Serve embedded web files
	webFS, err := fs.Sub(embed.WebFiles, "internal/assets/web")
	if err != nil {
		log.Printf("[INIT] ⚠️ Error loading embedded web files: %v", err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(webFS)))
		log.Println("[INIT] 🌐 Serving embedded web client")
	}

	p.httpServer = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  cfg.IdleTimeout,
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		log.Println("┌─────────────────────────────────────────────────────────────┐")
		log.Printf("│ 🎫 TICKET DAEMON READY - Environment: %-22s│", cfg.Name)
		log.Printf("│ 🔌 WebSocket: ws://%s/ws%-25s│", cfg.ListenAddr, "")
		log.Printf("│ 🌐 Dashboard: http://%s%-27s│", cfg.ListenAddr, "")
		log.Printf("│ 💚 Health:     http://%s/health%-20s│", cfg.ListenAddr, "")
		log.Println("└─────────────────────────────────────────────────────────────┘")

		if err := p.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Printf("[HTTP] ❌ Error starting HTTP server: %v", err)
		}
	}()

	return nil
}

// Stop stops the service gracefully
func (p *Program) Stop() error {
	log.Println("[STOP] 🛑 Service shutting down...")

	if p.printWorker != nil {
		p.printWorker.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if p.httpServer != nil {
		if err := p.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[STOP] ⚠️ HTTP shutdown error: %v", err)
		}
	}

	if p.wsServer != nil {
		p.wsServer.Shutdown()
	}

	close(p.quit)
	p.wg.Wait()

	uptime := time.Since(p.startTime)
	log.Printf("[STOP] ✅ Service stopped (uptime: %v)", uptime.Round(time.Second))
	return nil
}

func initLogging(envConfig config.Environment) error {
	logPath := envConfig.LogPath(os.Getenv("PROGRAMDATA"))
	logDir := filepath.Dir(logPath)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	if err := InitLogger(logPath, envConfig.Verbose); err != nil {
		return err
	}

	log.Printf("[INIT] 📁 Log file: %s", logPath)
	return nil
}
