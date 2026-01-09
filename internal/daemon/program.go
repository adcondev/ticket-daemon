// Package daemon contiene la lógica del servicio de Windows.
package daemon

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/judwhite/go-svc"

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
		DefaultPrinter: "", // Must be specified in document
	},
	"test": {
		Name:           "TEST/DEV",
		ServiceName:    serviceNameTest,
		ListenAddr:     "localhost:8766",
		Verbose:        true,
		DefaultPrinter: "80mm EC-PM-80250", // Default for testing
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
}

// Init initializes the service (logging, directories, etc.)
func (p *Program) Init(env svc.Environment) error {
	envConfig := GetEnvConfig()

	// Initialize logging
	if err := initLogging(envConfig); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║   TICKET DAEMON - Servicio de Impresión POS               ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Printf("[i] Iniciando Servicio - Ambiente: %s", envConfig.Name)
	log.Printf("[i] Build: %s %s", BuildDate, BuildTime)

	return nil
}

// Start starts the service
func (p *Program) Start() error {
	p.quit = make(chan struct{})
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

	// Start worker
	p.printWorker.Start()

	// Create HTTP server
	mux := http.NewServeMux()

	// WebSocket endpoint
	mux.HandleFunc("/ws", p.wsServer.HandleWebSocket)

	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		current, capacity := p.wsServer.QueueStatus()
		workerStats := p.printWorker.Stats()
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","queue": {"current":%d,"capacity":%d},"worker":{"running":%t}}`,
			current, capacity, workerStats.IsRunning)
	})

	// Static files for test client
	mux.Handle("/", http.FileServer(http.Dir("web")))

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

		log.Printf("[i] Servidor TICKET DAEMON - Ambiente: %s", envConfig.Name)
		log.Printf("[i] Build: %s %s", BuildDate, BuildTime)
		log.Printf("[i] WebSocket activo en ws://%s/ws", envConfig.ListenAddr)
		log.Printf("[i] Health check en http://%s/health", envConfig.ListenAddr)
		log.Printf("[i] Impresora por defecto: %s", defaultOrNone(envConfig.DefaultPrinter))

		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[X] Error al iniciar servidor HTTP: %v", err)
		}
	}()

	return nil
}

// Stop stops the service gracefully
func (p *Program) Stop() error {
	log.Println("[.] Servicio deteniéndose...")

	// Stop worker first (drain queue or timeout)
	if p.printWorker != nil {
		p.printWorker.Stop()
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if p.httpServer != nil {
		if err := p.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[!] Error durante shutdown HTTP: %v", err)
		}
	}

	// Close WebSocket server (notifies clients)
	if p.wsServer != nil {
		p.wsServer.Shutdown()
	}

	close(p.quit)
	p.wg.Wait()

	log.Println("[OK] Servicio detenido correctamente")
	return nil
}

// initLogging sets up file logging with rotation
func initLogging(envConfig EnvironmentConfig) error {
	// Configure log file path
	logDir := filepath.Join(os.Getenv("PROGRAMDATA"), envConfig.ServiceName)
	logPath := filepath.Join(logDir, envConfig.ServiceName+".log")

	// Create directory if not exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// Initialize logger
	if err := InitLogger(logPath, envConfig.Verbose); err != nil {
		return err
	}

	log.Printf("[i] Logs en: %s", logPath)
	log.Printf("[i] Verbose: %v", envConfig.Verbose)

	return nil
}

func defaultOrNone(s string) string {
	if s == "" {
		return "(ninguna - debe especificarse en documento)"
	}
	return s
}
