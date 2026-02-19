package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/adcondev/poster/pkg/connection"
	"github.com/adcondev/ticket-daemon/internal/posprinter"
)

type mockPrinterDiscovery struct{}

func (m *mockPrinterDiscovery) GetPrinters(_ bool) ([]connection.PrinterDetail, error) {
	return []connection.PrinterDetail{}, nil
}

func (m *mockPrinterDiscovery) GetSummary() posprinter.Summary {
	return posprinter.Summary{}
}

func TestWebSocketOrigin(t *testing.T) {
	// 1. Test Restricted Origin (Default behavior / Explicit Allow)
	t.Run("Restricted Origin", func(t *testing.T) {
		// Create server with specific allowed origin
		cfg := Config{
			QueueSize:      10,
			AllowedOrigins: []string{"http://good.com"},
		}
		discovery := &mockPrinterDiscovery{}
		srv := NewServer(cfg, discovery)
		defer srv.Shutdown()

		ts := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
		defer ts.Close()

		u := "ws" + ts.URL[4:]

		// Case A: Connection from Allowed Origin
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		opts := &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": []string{"http://good.com"},
			},
		}

		conn, resp, err := websocket.Dial(ctx, u, opts)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			t.Fatalf("Connection from good.com failed: %v", err)
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")

		// Case B: Connection from Disallowed Origin
		optsBad := &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": []string{"http://evil.com"},
			},
		}

		_, respBad, err := websocket.Dial(ctx, u, optsBad)
		if respBad != nil && respBad.Body != nil {
			_ = respBad.Body.Close()
		}
		if err == nil {
			t.Fatalf("Connection from evil.com succeeded (should fail)")
		}
	})

	// 2. Test Same Origin Enforcement (When AllowedOrigins is empty/nil)
	t.Run("Same Origin Enforcement", func(t *testing.T) {
		cfg := Config{
			QueueSize:      10,
			AllowedOrigins: nil, // Enforce same origin
		}
		discovery := &mockPrinterDiscovery{}
		srv := NewServer(cfg, discovery)
		defer srv.Shutdown()

		ts := httptest.NewServer(http.HandlerFunc(srv.HandleWebSocket))
		defer ts.Close()

		u := "ws" + ts.URL[4:]

		// Case A: Connection from Same Origin (Default behavior of Dial)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// websocket.Dial sets Origin to the URL's host by default, mimicking a same-origin request
		conn, resp, err := websocket.Dial(ctx, u, nil)
		if resp != nil && resp.Body != nil {
			_ = resp.Body.Close()
		}
		if err != nil {
			t.Fatalf("Connection from same origin failed: %v", err)
		}
		_ = conn.Close(websocket.StatusNormalClosure, "")

		// Case B: Connection from Different Origin
		optsBad := &websocket.DialOptions{
			HTTPHeader: http.Header{
				"Origin": []string{"http://external-site.com"},
			},
		}
		_, respBad, err := websocket.Dial(ctx, u, optsBad)
		if respBad != nil && respBad.Body != nil {
			_ = respBad.Body.Close()
		}
		if err == nil {
			t.Fatalf("Connection from external-site.com succeeded (should fail)")
		}
	})
}
