package config

import (
	"testing"
	"time"
)

func TestGetEnvironment_DirectKeys(t *testing.T) {
	tests := []struct {
		env         string
		wantName    string
		wantService string
		wantAddr    string
	}{
		{"remote", "REMOTO", "R2k_TicketServicio_Remoto", "0.0.0.0:8766"},
		{"local", "LOCAL", "R2k_TicketServicio_Local", "localhost:8766"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := GetEnvironment(tt.env)
			if cfg.Name != tt.wantName {
				t.Errorf("GetEnvironment(%q).Name = %q, want %q", tt.env, cfg.Name, tt.wantName)
			}
			if cfg.ServiceName != tt.wantService {
				t.Errorf("GetEnvironment(%q).ServiceName = %q, want %q", tt.env, cfg.ServiceName, tt.wantService)
			}
			if cfg.ListenAddr != tt.wantAddr {
				t.Errorf("GetEnvironment(%q).ListenAddr = %q, want %q", tt.env, cfg.ListenAddr, tt.wantAddr)
			}
		})
	}
}

func TestGetEnvironment_Aliases(t *testing.T) {
	tests := []struct {
		env         string
		wantName    string
		wantService string
		wantAddr    string
	}{
		// "test" should map to "local"
		{"test", "LOCAL", "R2k_TicketServicio_Local", "localhost:8766"},
		// "prod" should map to "remote"
		{"prod", "REMOTO", "R2k_TicketServicio_Remoto", "0.0.0.0:8766"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := GetEnvironment(tt.env)
			if cfg.Name != tt.wantName {
				t.Errorf("GetEnvironment(%q).Name = %q, want %q", tt.env, cfg.Name, tt.wantName)
			}
			if cfg.ServiceName != tt.wantService {
				t.Errorf("GetEnvironment(%q).ServiceName = %q, want %q", tt.env, cfg.ServiceName, tt.wantService)
			}
			if cfg.ListenAddr != tt.wantAddr {
				t.Errorf("GetEnvironment(%q).ListenAddr = %q, want %q", tt.env, cfg.ListenAddr, tt.wantAddr)
			}
		})
	}
}

func TestGetEnvironment_UnknownFallsBackToRemote(t *testing.T) {
	cfg := GetEnvironment("unknown")
	if cfg.Name != "REMOTO" {
		t.Errorf("GetEnvironment(%q).Name = %q, want %q", "unknown", cfg.Name, "REMOTO")
	}
	if cfg.ServiceName != "R2k_TicketServicio_Remoto" {
		t.Errorf("GetEnvironment(%q).ServiceName = %q, want %q", "unknown", cfg.ServiceName, "R2k_TicketServicio_Remoto")
	}
}

func TestGetEnvironment_LogNaming(t *testing.T) {
	// Verify naming convention: R2k_<<Daemon>><<Type>>_<<Env>>
	tests := []struct {
		env         string
		wantService string
	}{
		{"test", "R2k_TicketServicio_Local"},
		{"prod", "R2k_TicketServicio_Remoto"},
		{"local", "R2k_TicketServicio_Local"},
		{"remote", "R2k_TicketServicio_Remoto"},
	}
	for _, tt := range tests {
		t.Run(tt.env, func(t *testing.T) {
			cfg := GetEnvironment(tt.env)
			// The log file should be named ServiceName + ".log"
			expectedLog := cfg.ServiceName + ".log"
			wantLog := tt.wantService + ".log"
			if expectedLog != wantLog {
				t.Errorf("Log name for env %q = %q, want %q", tt.env, expectedLog, wantLog)
			}
		})
	}
}

func TestEnvironments_Timeouts(t *testing.T) {
	local := GetEnvironment("local")
	if local.ReadTimeout != 30*time.Second {
		t.Errorf("local ReadTimeout = %v, want 30s", local.ReadTimeout)
	}

	remote := GetEnvironment("remote")
	if remote.ReadTimeout != 15*time.Second {
		t.Errorf("remote ReadTimeout = %v, want 15s", remote.ReadTimeout)
	}
}
