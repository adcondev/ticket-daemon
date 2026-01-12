package daemon

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/judwhite/go-svc"

	"github.com/adcondev/ticket-daemon/internal/server"

	"github.com/adcondev/ticket-daemon/internal/assets"
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
}

// Init initializes the service
func (p *Program) Init(env svc.Environment) error {
	envConfig := GetEnvConfig()

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
		fmt.Fprintf(w, `{"status":"ok","queue": {"current":%d,"capacity":%d},"worker":{"running":%t},"build":{"env":"%s","date":"%s"}}`,
			current, capacity, workerStats.IsRunning, BuildEnvironment, BuildDate)
	})

	// Serve embedded web files
	webFS, err := fs.Sub(assets.WebFiles, "web")
	if err != nil {
		log.Printf("[! ] Error loading embedded web files: %v", err)
	} else {
		mux.Handle("/", http.FileServer(http.FS(webFS)))
		log.Println("[i] Serving embedded web client")
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

		log.Printf("[i] Servidor TICKET DAEMON - Ambiente: %s", envConfig.Name)
		log.Printf("[i] WebSocket: ws://%s/ws", envConfig.ListenAddr)
		log.Printf("[i] Test client: http://%s", envConfig.ListenAddr)
		log.Printf("[i] Health: http://%s/health", envConfig.ListenAddr)

		if err := p.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[X] Error al iniciar servidor HTTP: %v", err)
		}
	}()

	return nil
}

// Stop stops the service gracefully
func (p *Program) Stop() error {
	log.Println("[.] Servicio deteniéndose...")

	if p.printWorker != nil {
		p.printWorker.Stop()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if p.httpServer != nil {
		if err := p.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[! ] Error durante shutdown HTTP: %v", err)
		}
	}

	if p.wsServer != nil {
		p.wsServer.Shutdown()
	}

	close(p.quit)
	p.wg.Wait()

	log.Println("[OK] Servicio detenido correctamente")
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

	log.Printf("[i] Logs en: %s", logPath)
	return nil
}
