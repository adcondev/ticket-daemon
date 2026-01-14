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

	"github.com/adcondev/ticket-daemon/internal/assets"
	"github.com/adcondev/ticket-daemon/internal/server"
	"github.com/adcondev/ticket-daemon/internal/worker"
)

// Build variables, injected at compile time
var (
	BuildEnvironment = "prod"
	BuildDate        = "unknown"
	BuildTime        = "unknown"
)

const (
	serviceName     = "TicketServicio"
	serviceNameTest = "TicketServicioTest"
)

// EnvironmentConfig holds ALL environment-specific configuration
type EnvironmentConfig struct {
	// Identificación
	Name        string
	ServiceName string

	// Red
	ListenAddr   string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Cola
	QueueCapacity int

	// Logging
	Verbose bool

	// Impresora
	DefaultPrinter string
}

var envConfigs = map[string]EnvironmentConfig{
	"prod": {
		Name:           "PRODUCCIÓN",
		ServiceName:    serviceName,
		ListenAddr:     "0.0.0.0:8766",
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		QueueCapacity:  100,
		Verbose:        false,
		DefaultPrinter: "",
	},
	"test": {
		Name:           "TEST/DEV",
		ServiceName:    serviceNameTest,
		ListenAddr:     "localhost:8766",
		ReadTimeout:    30 * time.Second, // Más tiempo para debugging
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		QueueCapacity:  50, // Menor para detectar problemas rápido
		Verbose:        true,
		DefaultPrinter: "58mm PT-210",
	},
}

// GetEnvConfig returns the current environment configuration
func GetEnvConfig() EnvironmentConfig {
	if config, ok := envConfigs[BuildEnvironment]; ok {
		return config
	}
	return envConfigs["prod"]
}

// Program implements svc.Service interface
type Program struct {
	wg          sync.WaitGroup
	quit        chan struct{}
	httpServer  *http.Server
	wsServer    *server.Server
	printWorker *worker.Worker
	startTime   time.Time
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

	// Initialize WebSocket server
	p.wsServer = server.NewServer(server.Config{
		QueueSize: cfg.QueueCapacity,
	})

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
			Build: BuildInfo{
				Env:  BuildEnvironment,
				Date: BuildDate,
				Time: BuildTime,
			},
			Uptime: int(time.Since(p.startTime).Seconds()),
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// json.NewEncoder es más eficiente que Marshal + Write
		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, `{"error":"encoding failed"}`, http.StatusInternalServerError)
		}
	})

	// Serve embedded web files
	webFS, err := fs.Sub(assets.WebFiles, "web")
	if err != nil {
		log.Printf("[INIT] ⚠️ Error loading embedded web files: %v", err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(webFS)))
		log.Println("[INIT] 🌐 Serving embedded web client")
	}

	p.httpServer = &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
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

func initLogging(envConfig EnvironmentConfig) error {
	logDir := filepath.Join(os.Getenv("PROGRAMDATA"), envConfig.ServiceName)
	logPath := filepath.Join(logDir, envConfig.ServiceName+".log")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	if err := InitLogger(logPath, envConfig.Verbose); err != nil {
		return err
	}

	log.Printf("[INIT] 📁 Log file: %s", logPath)
	return nil
}
