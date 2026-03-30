package daemon

import (
	"testing"
	"time"
)

func TestNewPrinterDiscovery(t *testing.T) {
	ttl := 10 * time.Second
	pd := NewPrinterDiscovery(ttl)
	if pd.cacheTTL != ttl {
		t.Errorf("expected cacheTTL %v, got %v", ttl, pd.cacheTTL)
	}
}
