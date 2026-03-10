package workererrors

import (
	"errors"
	"testing"
)

func TestExtractUserFriendlyError(t *testing.T) {
	tests := []struct {
		name     string
		input    error
		expected string
	}{
		// Specific Error Mappings
		{
			name:     "Version required",
			input:    errors.New("validation failed: version is required"),
			expected: "VALIDATION: Missing 'version' field",
		},
		{
			name:     "Profile model required",
			input:    errors.New("profile.model is required"),
			expected: "VALIDATION: Missing 'profile.model' field",
		},
		{
			name:     "At least one command",
			input:    errors.New("document must contain at least one command"),
			expected: "VALIDATION: Document must contain at least one command",
		},
		{
			name:     "Invalid paper width",
			input:    errors.New("invalid paper_width: 100"),
			expected: "VALIDATION: Invalid paper width (use 58 or 80)",
		},
		{
			name:     "Invalid DPI",
			input:    errors.New("invalid dpi provided"),
			expected: "VALIDATION: Invalid DPI value",
		},
		{
			name:     "Invalid version format",
			input:    errors.New("invalid version format"),
			expected: "VALIDATION: Invalid version format (use X.Y pattern)",
		},
		{
			name:     "Printer connection error",
			input:    errors.New("error conectando a impresora: connection refused"),
			expected: "PRINTER: Cannot connect - check if printer is installed",
		},
		{
			name:     "Printer name not specified",
			input:    errors.New("nombre de impresora no especificado"),
			expected: "PRINTER: No printer name specified in profile.model",
		},
		{
			name:     "QR data empty",
			input:    errors.New("QR data cannot be empty"),
			expected: "QR:  Data cannot be empty",
		},
		{
			name:     "QR data too long",
			input:    errors.New("QR data too long"),
			expected: "QR: Data exceeds maximum length",
		},
		{
			name:     "Barcode symbology required",
			input:    errors.New("barcode symbology is required"),
			expected: "BARCODE: Symbology type is required",
		},
		{
			name:     "Barcode data required",
			input:    errors.New("barcode data is required"),
			expected: "BARCODE: Data is required",
		},
		{
			name:     "Table overflow",
			input:    errors.New("table overflow"),
			expected: "TABLE:  Columns exceed paper width",
		},
		{
			name:     "Raw command empty",
			input:    errors.New("raw command cannot be empty"),
			expected: "RAW: Command hex cannot be empty",
		},
		{
			name:     "Unsafe command blocked",
			input:    errors.New("unsafe command blocked"),
			expected: "RAW:  Blocked by safe_mode - potentially dangerous command",
		},
		{
			name:     "Failed to load image",
			input:    errors.New("failed to load image"),
			expected: "IMAGE: Invalid or corrupted base64 data",
		},
		{
			name:     "Invalid QR correction level",
			input:    errors.New("invalid QR correction level"),
			expected: "QR:  Invalid correction level (use L, M, Q, or H)",
		},
		{
			name:     "Unknown command type",
			input:    errors.New("unknown command type"),
			expected: "COMMAND:  Unknown command type",
		},

		// Categorization Logic
		{
			name:     "Document invalid (categorization)",
			input:    errors.New("documento inválido: field X is wrong"),
			expected: "VALIDATION: field X is wrong",
		},
		{
			name:     "Error parsing (categorization)",
			input:    errors.New("error parseando JSON"),
			expected: "JSON: Invalid document structure",
		},
		{
			name:     "Error executing (categorization)",
			input:    errors.New("error ejecutando: printer failed"),
			expected: "EXECUTION: printer failed",
		},

		// Fallback Logic
		{
			name:     "Fallback with clean error message",
			input:    errors.New("some random error"),
			expected: "ERROR: some random error",
		},
		{
			name:     "Fallback with prefix removal",
			input:    errors.New("error creando servicio de impresión: specific service error"),
			expected: "ERROR: specific service error", // Verify prefix removal logic
		},
		{
			name:     "Nested error",
			input:    errors.New("outer error: inner error"),
			expected: "ERROR: outer error: inner error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUserFriendlyError(tt.input)
			if got != tt.expected {
				t.Errorf("ExtractUserFriendlyError() = %v, want %v", got, tt.expected)
			}
		})
	}
}
