package daemon

import (
	"context"
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

// EnvironmentConfig holds environment-specific configuration
type EnvironmentConfig struct {
	Name           string
	ServiceName    string
	ListenAddr     string
	Verbose        bool
	DefaultPrinter string
}

var envConfigs = map[string]EnvironmentConfig{
	"prod": {
		Name:           "PRODUCCIÓN",
		ServiceName:    serviceName,
		ListenAddr:     "0.0.0.0:8766",
		Verbose:        false,
		DefaultPrinter: "",
	},
	"test": {
		Name:           "TEST/DEV",
		ServiceName:    serviceNameTest,
		ListenAddr:     "localhost:8766",
		Verbose:        true,
		DefaultPrinter: "80mm EC-PM-80250",
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
	envConfig := GetEnvConfig()

	// Initialize WebSocket server
	p.wsServer = server.NewServer(server.Config{
		QueueSize: 100,
	})

	// Initialize print worker
	p.printWorker = worker.NewWorker(
		p.wsServer.JobQueue(),
		p.wsServer,
		worker.Config{
			DefaultPrinter: envConfig.DefaultPrinter,
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
		workerStats := p.printWorker.Stats()
		uptime := time.Since(p.startTime)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		// Enhanced health response
		response := fmt.Sprintf(`{
  "status": "ok",
  "queue": {
    "current": %d,
    "capacity": %d,
    "utilization": %.1f
  },
  "worker": {
    "running": %t,
    "jobs_processed": %d,
    "jobs_failed": %d
  },
  "build": {
    "env": "%s",
    "date": "%s",
    "time": "%s"
  },
  "uptime_seconds": %d
}`,
			current,
			capacity,
			float64(current)/float64(capacity)*100,
			workerStats.IsRunning,
			workerStats.JobsProcessed,
			workerStats.JobsFailed,
			BuildEnvironment,
			BuildDate,
			BuildTime,
			int(uptime.Seconds()),
		)

		fmt.Fprint(w, response)
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
		Addr:         envConfig.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()

		log.Println("┌─────────────────────────────────────────────────────────────┐")
		log.Printf("│ 🎫 TICKET DAEMON READY - Environment: %-22s│", envConfig.Name)
		log.Printf("│ 🔌 WebSocket: ws://%s/ws%-25s│", envConfig.ListenAddr, "")
		log.Printf("│ 🌐 Dashboard: http://%s%-27s│", envConfig.ListenAddr, "")
		log.Printf("│ 💚 Health:     http://%s/health%-20s│", envConfig.ListenAddr, "")
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
