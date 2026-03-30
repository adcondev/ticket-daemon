package config

import (
	"path/filepath"
	"testing"
)

func TestEnvironment_LogPath(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		programData string
		want        string
	}{
		{
			name:        "Standard path",
			serviceName: "TestService",
			programData: "/var/lib",
			want:        filepath.Join("/var/lib", "TestService", "TestService.log"),
		},
		{
			name:        "Empty program data",
			serviceName: "TestService",
			programData: "",
			want:        filepath.Join("TestService", "TestService.log"),
		},
		{
			name:        "Program data with trailing separator",
			serviceName: "TestService",
			programData: "/var/lib/",
			want:        filepath.Join("/var/lib", "TestService", "TestService.log"),
		},
		{
			name:        "Service name with spaces",
			serviceName: "Test Service",
			programData: "/var/lib",
			want:        filepath.Join("/var/lib", "Test Service", "Test Service.log"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := Environment{
				ServiceName: tt.serviceName,
			}
			if got := e.LogPath(tt.programData); got != tt.want {
				t.Errorf("Environment.LogPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
