package daemon

import (
	"github.com/adcondev/ticket-daemon/internal/printer"
)

// HealthResponse representa el estado de salud del servicio de impresión.
type HealthResponse struct {
	Status   string          `json:"status"`
	Queue    QueueStatus     `json:"queue"`
	Worker   WorkerStatus    `json:"worker"`
	Printers printer.Summary `json:"printers"` // NEW
	Build    BuildInfo       `json:"build"`
	Uptime   int             `json:"uptime_seconds"`
}

// QueueStatus representa el estado de la cola de impresión.
type QueueStatus struct {
	Current     int     `json:"current"`
	Capacity    int     `json:"capacity"`
	Utilization float64 `json:"utilization"`
}

// WorkerStatus representa el estado del trabajador de impresión.
type WorkerStatus struct {
	Running       bool  `json:"running"`
	JobsProcessed int64 `json:"jobs_processed"`
	JobsFailed    int64 `json:"jobs_failed"`
}

// BuildInfo contiene información sobre la compilación del servicio.
type BuildInfo struct {
	Env  string `json:"env"`
	Date string `json:"date"`
	Time string `json:"time"`
}
