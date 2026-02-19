package worker

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/adcondev/ticket-daemon/internal/server"
	"github.com/coder/websocket"
)

// mockSlowNotifier simulates a slow network connection
type mockSlowNotifier struct {
	delay time.Duration
}

func (m *mockSlowNotifier) NotifyClient(_ *websocket.Conn, response server.Response) error {
	time.Sleep(m.delay)
	return nil
}

func TestWorkerBlockingNotification(t *testing.T) {
	// Setup
	jobCount := 5
	notifier := &mockSlowNotifier{delay: 200 * time.Millisecond} // 200ms delay per notification

	// Create job queue
	jobQueue := make(chan *server.PrintJob, jobCount)

	// Create worker
	config := Config{DefaultPrinter: "test"}
	w := NewWorker(jobQueue, notifier, config)

	// Start worker
	w.Start()
	defer w.Stop()

	// Create dummy connection (we need a non-nil pointer)
	dummyConn := &websocket.Conn{}

	// Prepare jobs
	for j := 0; j < jobCount; j++ {
		job := &server.PrintJob{
			ID:         "test-job",
			ClientConn: dummyConn,
			Document:   json.RawMessage("{}"), // Invalid document to trigger failure but still notify
			ReceivedAt: time.Now(),
		}
		jobQueue <- job
	}

	start := time.Now()

	// Wait for processing
	deadline := time.Now().Add(5 * time.Second)
	for {
		stats := w.Stats()
		if stats.JobsProcessed+stats.JobsFailed >= int64(jobCount) {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("Timeout waiting for jobs to process. Processed: %d, Failed: %d", stats.JobsProcessed, stats.JobsFailed)
		}
		time.Sleep(10 * time.Millisecond)
	}

	duration := time.Since(start)

	// With blocking notification: 5 jobs * 200ms = 1000ms (1s)
	// We expect it to take at least 1s.
	if duration > 500*time.Millisecond {
		t.Errorf("Expected duration < 500ms (async), got %v", duration)
	} else {
		t.Logf("Blocking verified: Duration %v for %d jobs", duration, jobCount)
	}
}
