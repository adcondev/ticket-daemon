package config

import (
	"path/filepath"
	"testing"
)

func TestGetEnvironment(t *testing.T) {
	// Table-driven test cases
	tests := []struct {
		name          string
		inputEnv      string
		expectedName  string
		expectedAddr  string
		expectedQCap  int
		expectDefault bool // If true, we expect the fallback (local) config
	}{
		{
			name:         "Get local environment",
			inputEnv:     "local",
			expectedName: "LOCAL",
			expectedAddr: "localhost:" + ServerPort,
			expectedQCap: 50,
		},
		{
			name:         "Get remote environment",
			inputEnv:     "remote",
			expectedName: "REMOTO",
			expectedAddr: "0.0.0.0:" + ServerPort,
			expectedQCap: 50,
		},
		{
			name:          "Get unknown environment (defaults to local)",
			inputEnv:      "unknown_env",
			expectedName:  "LOCAL",
			expectedAddr:  "localhost:" + ServerPort,
			expectedQCap:  50,
			expectDefault: true,
		},
		{
			name:          "Get empty environment (defaults to local)",
			inputEnv:      "",
			expectedName:  "LOCAL",
			expectedAddr:  "localhost:" + ServerPort,
			expectedQCap:  50,
			expectDefault: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEnvironment(tt.inputEnv)

			// Verify key fields
			if got.Name != tt.expectedName {
				t.Errorf("GetEnvironment(%q).Name = %q; want %q", tt.inputEnv, got.Name, tt.expectedName)
			}
			if got.ListenAddr != tt.expectedAddr {
				t.Errorf("GetEnvironment(%q).ListenAddr = %q; want %q", tt.inputEnv, got.ListenAddr, tt.expectedAddr)
			}
			if got.QueueCapacity != tt.expectedQCap {
				t.Errorf("GetEnvironment(%q).QueueCapacity = %d; want %d", tt.inputEnv, got.QueueCapacity, tt.expectedQCap)
			}

			// Verify timeout settings are reasonable (not zero)
			if got.ReadTimeout == 0 {
				t.Errorf("GetEnvironment(%q).ReadTimeout is 0; expected non-zero duration", tt.inputEnv)
			}
			if got.WriteTimeout == 0 {
				t.Errorf("GetEnvironment(%q).WriteTimeout is 0; expected non-zero duration", tt.inputEnv)
			}

			// Specific check for local environment details if expected
			if tt.expectDefault {
				// Verify it matches the 'local' config exactly
				localCfg := environments["local"]
				if got.Name != localCfg.Name {
					t.Errorf("GetEnvironment(%q) did not return local config as default", tt.inputEnv)
				}
			}
		})
	}
}

func TestEnvironment_LogPath(t *testing.T) {
	// Quick test for the LogPath method as well
	env := Environment{
		ServiceName: "TestService",
	}
	programData := "/var/lib"
	expected := filepath.Join(programData, "TestService", "TestService.log")

	got := env.LogPath(programData)

	if got != expected {
		t.Errorf("LogPath(%q) = %q; want %q", programData, got, expected)
	}
}
