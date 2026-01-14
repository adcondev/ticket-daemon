package daemon

type HealthResponse struct {
	Status string       `json:"status"`
	Queue  QueueStatus  `json:"queue"`
	Worker WorkerStatus `json:"worker"`
	Build  BuildInfo    `json:"build"`
	Uptime int          `json:"uptime_seconds"`
}

type QueueStatus struct {
	Current     int     `json:"current"`
	Capacity    int     `json:"capacity"`
	Utilization float64 `json:"utilization"`
}

type WorkerStatus struct {
	Running       bool  `json:"running"`
	JobsProcessed int64 `json:"jobs_processed"`
	JobsFailed    int64 `json:"jobs_failed"`
}

type BuildInfo struct {
	Env  string `json:"env"`
	Date string `json:"date"`
	Time string `json:"time"`
}
