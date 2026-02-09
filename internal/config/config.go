package config

import (
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

// Environments defines available deployment configurations
var Environments = map[string]Environment{
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
	if cfg, ok := Environments[env]; ok {
		return cfg
	}
	if alias, ok := envAliases[env]; ok {
		if cfg, ok := Environments[alias]; ok {
			return cfg
		}
	}
	return Environments["remote"]
}
