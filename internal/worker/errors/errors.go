package workererrors

import (
	"fmt"
	"strings"
)

// ExtractUserFriendlyError creates a clean error message for the UI
func ExtractUserFriendlyError(err error) string {
	errStr := err.Error()

	// Common error patterns and their friendly messages
	errorMappings := []struct {
		pattern string
		message string
	}{
		{"version is required", "VALIDATION: Missing 'version' field"},
		{"profile.model is required", "VALIDATION: Missing 'profile.model' field"},
		{"at least one command", "VALIDATION: Document must contain at least one command"},
		{"invalid paper_width", "VALIDATION: Invalid paper width (use 58 or 80)"},
		{"invalid dpi", "VALIDATION: Invalid DPI value"},
		{"invalid version format", "VALIDATION: Invalid version format (use X.Y pattern)"},
		{"error conectando a impresora", "PRINTER: Cannot connect - check if printer is installed"},
		{"nombre de impresora no especificado", "PRINTER: No printer name specified in profile.model"},
		{"QR data cannot be empty", "QR:  Data cannot be empty"},
		{"QR data too long", "QR: Data exceeds maximum length"},
		{"barcode symbology is required", "BARCODE: Symbology type is required"},
		{"barcode data is required", "BARCODE: Data is required"},
		{"table overflow", "TABLE:  Columns exceed paper width"},
		{"raw command cannot be empty", "RAW: Command hex cannot be empty"},
		{"unsafe command blocked", "RAW:  Blocked by safe_mode - potentially dangerous command"},
		{"failed to load image", "IMAGE: Invalid or corrupted base64 data"},
		{"invalid QR correction level", "QR:  Invalid correction level (use L, M, Q, or H)"},
		{"unknown command type", "COMMAND:  Unknown command type"},
	}

	// Check for matching patterns
	for _, mapping := range errorMappings {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(mapping.pattern)) {
			return mapping.message
		}
	}

	// Categorize by error source
	if strings.Contains(errStr, "documento inválido") {
		return fmt.Sprintf("VALIDATION: %s", extractInnerError(errStr))
	}
	if strings.Contains(errStr, "error parseando") {
		return "JSON: Invalid document structure"
	}
	if strings.Contains(errStr, "error ejecutando") {
		return fmt.Sprintf("EXECUTION: %s", extractInnerError(errStr))
	}

	// Fallback:  return cleaned error
	return fmt.Sprintf("ERROR: %s", cleanErrorMessage(errStr))
}

// extractInnerError gets the innermost error message
func extractInnerError(errStr string) string {
	// Find the last colon-separated segment
	parts := strings.Split(errStr, ": ")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return errStr
}

// cleanErrorMessage removes verbose prefixes
func cleanErrorMessage(errStr string) string {
	// Remove common prefixes
	prefixes := []string{
		"error parseando documento:  ",
		"documento inválido: ",
		"error ejecutando documento: ",
		"error creando servicio de impresión: ",
	}
	result := errStr
	for _, prefix := range prefixes {
		result = strings.TrimPrefix(result, prefix)
	}
	return result
}
