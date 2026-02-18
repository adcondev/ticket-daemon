package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/judwhite/go-svc"

	embed "github.com/adcondev/ticket-daemon"
	"github.com/adcondev/ticket-daemon/internal/auth"
	"github.com/adcondev/ticket-daemon/internal/config"
	"github.com/adcondev/ticket-daemon/internal/server"
	"github.com/adcondev/ticket-daemon/internal/worker"
)

// GetEnvConfig returns the current environment configuration
func GetEnvConfig() config.Environment {
	return config.GetEnvironment(config.BuildEnvironment)
}

// Program implements svc.Service interface
type Program struct {
	wg               sync.WaitGroup
	quit             chan struct{}
	ctx              context.Context
	cancel           context.CancelFunc
	httpServer       *http.Server
	wsServer         *server.Server
	printWorker      *worker.Worker
	authMgr          *auth.Manager
	startTime        time.Time
	printerDiscovery *PrinterDiscovery
}

// Init initializes the service
func (p *Program) Init(_ svc.Environment) error {
	envConfig := GetEnvConfig()

	if err := initLogging(envConfig); err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	log.Println("╔════════════════════════════════════════════════════════════╗")
	log.Println("║   🎫 TICKET DAEMON - POS Print Service                     ║")
	log.Println("╚════════════════════════════════════════════════════════════╝")
	log.Printf("[INIT] 🚀 Starting service - Environment: %s", envConfig.Name)
	log.Printf("[INIT] 📅 Build: %s %s", config.BuildDate, config.BuildTime)

	return nil
}

// Start starts the service
func (p *Program) Start() error {
	p.quit = make(chan struct{})
	p.startTime = time.Now()
	p.ctx, p.cancel = context.WithCancel(context.Background())
	cfg := GetEnvConfig()

	// Initialize auth manager (bound to service context for clean shutdown)
	p.authMgr = auth.NewManager(p.ctx)

	p.printerDiscovery = NewPrinterDiscovery()
	p.printerDiscovery.LogStartupDiagnostics()

	// Initialize WebSocket server
	p.wsServer = server.NewServer(server.Config{QueueSize: cfg.QueueCapacity}, p.printerDiscovery)

	// Initialize print worker
	p.printWorker = worker.NewWorker(
		p.wsServer.JobQueue(),
		p.wsServer,
		worker.Config{DefaultPrinter: cfg.DefaultPrinter},
	)
	p.printWorker.Start()

	// Setup embedded filesystem
	webFS, err := fs.Sub(embed.WebFiles, "internal/assets/web")
	if err != nil {
		log.Fatalf("[FATAL] Error loading web assets: %v", err)
	}

	// Parse index.html as Go template for token injection
	indexBytes := readWebFile(webFS, "index.html")
	dashboardTmpl, err := template.New("dashboard").Parse(string(indexBytes))
	if err != nil {
		log.Fatalf("[FATAL] Error parsing index.html as template: %v", err)
	}

	// Read login.html
	loginHTML := readWebFile(webFS, "login.html")

	// Health handler (closure capturing program state)
	healthHandler := func(w http.ResponseWriter, _ *http.Request) {
		current, capacity := p.wsServer.QueueStatus()
		if capacity == 0 {
			current = 0
		}
		stats := p.printWorker.Stats()

		var utilization float64
		if capacity > 0 {
			utilization = float64(current) / float64(capacity) * 100
		}

		response := HealthResponse{
			Status: "ok",
			Queue: QueueStatus{
				Current:     current,
				Capacity:    capacity,
				Utilization: utilization,
			},
			Worker: WorkerStatus{
				Running:       stats.IsRunning,
				JobsProcessed: stats.JobsProcessed,
				JobsFailed:    stats.JobsFailed,
			},
			Printers: p.printerDiscovery.GetSummary(),
			Build: BuildInfo{
				Env:  config.BuildEnvironment,
				Date: config.BuildDate,
				Time: config.BuildTime,
			},
			Uptime: int(time.Since(p.startTime).Seconds()),
		}

		if response.Printers.Status == "error" {
			response.Status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		_ = json.NewEncoder(w).Encode(response)
	}

	// Create HTTP mux with auth boundaries
	mux := http.NewServeMux()

	// ── PUBLIC ROUTES (no auth required) ─────────────────────
	mux.Handle("/css/", http.FileServer(http.FS(webFS)))
	mux.Handle("/js/", http.FileServer(http.FS(webFS)))
	mux.HandleFunc("/login", serveLogin(p.authMgr, loginHTML))
	mux.HandleFunc("/auth/login", handleLogin(p.authMgr))
	mux.HandleFunc("/auth/logout", handleLogout(p.authMgr))
	mux.HandleFunc("/ws", p.wsServer.HandleWebSocket) // WS is public; token validates inside per-message
	mux.HandleFunc("/health", healthHandler)          // Health is public for monitoring tools

	// ── PROTECTED ROUTES (session required for dashboard) ────
	mux.HandleFunc("/", requireAuth(p.authMgr, serveDashboard(dashboardTmpl)))

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
		log.Printf("│ 🔐 Auth:       %-43v│", p.authMgr.Enabled())
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

	// 1. Cancel context (stops auth cleanup goroutine)
	p.cancel()

	// 2. Stop print worker
	if p.printWorker != nil {
		p.printWorker.Stop()
	}

	// 3. Graceful HTTP shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if p.httpServer != nil {
		if err := p.httpServer.Shutdown(ctx); err != nil {
			log.Printf("[STOP] ⚠️ HTTP shutdown error: %v", err)
		}
	}

	// 4. Shutdown WebSocket server
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

	if err := os.MkdirAll(logDir, 0750); err != nil {
		return err
	}

	if err := InitLogger(logPath, envConfig.Verbose); err != nil {
		return err
	}

	log.Printf("[INIT] 📁 Log file: %s", logPath)
	return nil
}

// requireAuth wraps a handler with session validation.
// If auth is disabled (no hash), all requests pass through.
func requireAuth(authMgr *auth.Manager, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authMgr.Enabled() {
			next(w, r)
			return
		}
		if !authMgr.GetSessionFromRequest(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// serveLogin returns a handler that serves the login page.
func serveLogin(authMgr *auth.Manager, loginHTML []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !authMgr.Enabled() {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		if authMgr.GetSessionFromRequest(r) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(loginHTML)
	}
}

// handleLogin processes POST /auth/login.
func handleLogin(authMgr *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ip := r.RemoteAddr
		if authMgr.IsLockedOut(ip) {
			log.Printf("[AUDIT] LOGIN_BLOCKED | IP=%s | reason=lockout", ip)
			http.Redirect(w, r, "/login?locked=1", http.StatusSeeOther)
			return
		}
		password := r.FormValue("password")
		if !authMgr.ValidatePassword(password) {
			authMgr.RecordFailedLogin(ip)
			log.Printf("[AUDIT] LOGIN_FAILED | IP=%s", ip)
			http.Redirect(w, r, "/login?error=1", http.StatusSeeOther)
			return
		}
		authMgr.ClearFailedLogins(ip)
		authMgr.SetSessionCookie(w)
		log.Printf("[AUDIT] LOGIN_SUCCESS | IP=%s", ip)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

// handleLogout clears the session.
func handleLogout(authMgr *auth.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authMgr.ClearSessionCookie(w)
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

// serveDashboard renders the dashboard template with token injection.
func serveDashboard(tmpl *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := struct{ AuthToken string }{AuthToken: config.AuthToken}
		if err := tmpl.Execute(w, data); err != nil {
			log.Printf("[X] Error rendering dashboard: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	}
}

// readWebFile reads a file from the embedded FS, fataling on error.
func readWebFile(webFS fs.FS, name string) []byte {
	data, err := fs.ReadFile(webFS, name)
	if err != nil {
		log.Fatalf("[FATAL] Error reading %s: %v", name, err)
	}
	return data
}
