package auth

import (
	"context"
	"encoding/base64"
	"testing"

	"github.com/adcondev/ticket-daemon/internal/config"
	"golang.org/x/crypto/bcrypt"
)

func TestValidatePassword(t *testing.T) {
	// Save the original config state to restore it after the test
	originalPasswordHashB64 := config.PasswordHashB64
	defer func() {
		config.PasswordHashB64 = originalPasswordHashB64
	}()

	// Helper to create a valid base64 bcrypt hash
	validPassword := "mysecurepassword"
	hash, err := bcrypt.GenerateFromPassword([]byte(validPassword), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("Failed to generate bcrypt hash: %v", err)
	}
	validHashB64 := base64.StdEncoding.EncodeToString(hash)

	// Context for manager
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	manager := NewManager(ctx)

	tests := []struct {
		name          string
		configHashB64 string
		inputPassword string
		expected      bool
	}{
		{
			name:          "Auth Disabled (Empty Hash)",
			configHashB64: "",
			inputPassword: "anypassword",
			expected:      true,
		},
		{
			name:          "Invalid Base64 Config Hash",
			configHashB64: "invalid_base64!",
			inputPassword: validPassword,
			expected:      false,
		},
		{
			name:          "Valid Base64 but Invalid Bcrypt Hash",
			configHashB64: base64.StdEncoding.EncodeToString([]byte("not_a_bcrypt_hash")),
			inputPassword: validPassword,
			expected:      false,
		},
		{
			name:          "Valid Hash and Correct Password",
			configHashB64: validHashB64,
			inputPassword: validPassword,
			expected:      true,
		},
		{
			name:          "Valid Hash but Incorrect Password",
			configHashB64: validHashB64,
			inputPassword: "wrongpassword",
			expected:      false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			config.PasswordHashB64 = tc.configHashB64
			result := manager.ValidatePassword(tc.inputPassword)
			if result != tc.expected {
				t.Errorf("ValidatePassword() for %q returned %v, expected %v", tc.name, result, tc.expected)
			}
		})
	}
}
