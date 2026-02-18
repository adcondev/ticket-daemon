// Package printer contains shared types to avoid import cycles.
package printer

// Summary PrinterSummary provides lightweight overview for health checks
type Summary struct {
	Status        string `json:"status"` // "ok", "warning", "error"
	DetectedCount int    `json:"detected_count"`
	ThermalCount  int    `json:"thermal_count"`
	DefaultName   string `json:"default_name,omitempty"`
}

// DetailDTO PrinterDetailDTO is the JSON response format for printer details
type DetailDTO struct {
	Name        string `json:"name"`
	Port        string `json:"port"`
	Driver      string `json:"driver"`
	Status      string `json:"status"`
	IsDefault   bool   `json:"is_default"`
	IsVirtual   bool   `json:"is_virtual"`
	PrinterType string `json:"printer_type"`
}
