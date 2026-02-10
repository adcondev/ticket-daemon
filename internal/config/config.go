package config

import (
	"path/filepath"
	"time"
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
		ServiceName:    "R2k_TicketServicio_Remoto",
		ListenAddr:     "0.0.0.0:8766",
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		IdleTimeout:    60 * time.Second,
		QueueCapacity:  100,
		Verbose:        false,
		DefaultPrinter: "",
	},
	"local": {
		Name:           "LOCAL",
		ServiceName:    "R2k_TicketServicio_Local",
		ListenAddr:     "localhost:8766",
		ReadTimeout:    30 * time.Second, // Más tiempo para debugging
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		QueueCapacity:  50, // Menor para detectar problemas rápido
		Verbose:        true,
		DefaultPrinter: "58mm PT-210",
	},
}

// envAliases maps build-time environment names to config keys.
// The Taskfile uses "test" for local and "prod" for remote builds.
var envAliases = map[string]string{
	"test": "local",
	"prod": "remote",
}

// GetEnvironment returns config for the specified environment.
// Supports aliases: "test" → "local", "prod" → "remote".
// Falls back to "remote" if unknown.
func GetEnvironment(env string) Environment {
	if cfg, ok := environments[env]; ok {
		return cfg
	}
	if alias, ok := envAliases[env]; ok {
		if cfg, ok := environments[alias]; ok {
			return cfg
		}
	}
	return environments["remote"]
}
