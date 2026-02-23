// Package config defines environment-specific settings for the R2k Ticket Servicio.
package config

import (
	"log"
	"path/filepath"
	"strings"
	"time"
)

// Build variables, injected at compile time
var (
	BuildEnvironment = "local"
	BuildDate        = "unknown"
	BuildTime        = "unknown"
	// ServiceName is used for logging and as part of the log file path.
	ServiceName = "R2k_TicketServicio_Unknown"
	// PasswordHashB64 is a base64-encoded bcrypt hash injected via ldflags.
	// If empty, dashboard authentication is disabled (dev mode).
	PasswordHashB64 = ""
	// AuthToken is injected via ldflags.
	// If empty, print job submissions are accepted without token validation.
	AuthToken = ""
	// ServerPort is the default port for the service, can be overridden by environment config.
	ServerPort = "8766"
	// AllowedOrigins is a comma-separated list of allowed origins injected via ldflags.
	// Example: "https://pos.example.com,http://localhost:*"
	AllowedOrigins = ""
)

// Environment holds environment-specific settings
type Environment struct {
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

	// Security
	AllowedOrigins []string
}

// LogPath returns the full log file path for this environment.
// Uses the convention: <programData>/<ServiceName>/<ServiceName>.log
func (e Environment) LogPath(programData string) string {
	return filepath.Join(programData, e.ServiceName, e.ServiceName+".log")
}

// environments defines available deployment configurations
var environments = map[string]Environment{
	"remote": {
		Name:           "REMOTO",
		ServiceName:    ServiceName,
		ListenAddr:     "0.0.0.0:" + ServerPort,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		QueueCapacity:  50,
		Verbose:        false,
		DefaultPrinter: "",
		// By default, restrict to localhost and file (Electron) for security
		AllowedOrigins: []string{"http://localhost:*", "https://localhost:*", "file://*"},
	},
	"local": {
		Name:           "LOCAL",
		ServiceName:    ServiceName,
		ListenAddr:     "localhost:" + ServerPort,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		QueueCapacity:  50,
		Verbose:        true,
		DefaultPrinter: "58mm PT-210",
		// Allow all in local dev mode for convenience, but can be overridden
		AllowedOrigins: []string{"*"},
	},
}

// GetEnvironment returns config for the specified environment.
func GetEnvironment(env string) Environment {
	cfg, ok := environments[env]
	if !ok {
		log.Printf("[!] Unknown environment '%s', defaulting to 'local'", env)
		cfg = environments["local"]
	}

	// Override allowed origins from ldflags if provided
	if AllowedOrigins != "" {
		cfg.AllowedOrigins = strings.Split(AllowedOrigins, ",")
	}

	return cfg
}
