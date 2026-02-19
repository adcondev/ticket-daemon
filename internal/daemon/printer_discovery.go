package daemon

import (
	"log"
	"sync"
	"time"

	"github.com/adcondev/poster/pkg/connection"
	"github.com/adcondev/ticket-daemon/internal/posprinter"
)

// PrinterDiscovery handles printer enumeration with caching
type PrinterDiscovery struct {
	cache       []connection.PrinterDetail
	lastRefresh time.Time
	cacheTTL    time.Duration
	mu          sync.RWMutex
}

// NewPrinterDiscovery creates a new discovery service
func NewPrinterDiscovery() *PrinterDiscovery {
	return &PrinterDiscovery{
		cacheTTL: 30 * time.Second,
	}
}

// GetPrinters returns cached printers or refreshes if stale
func (pd *PrinterDiscovery) GetPrinters(forceRefresh bool) ([]connection.PrinterDetail, error) {
	pd.mu.RLock()
	if !forceRefresh && time.Since(pd.lastRefresh) < pd.cacheTTL && pd.cache != nil {
		result := make([]connection.PrinterDetail, len(pd.cache))
		copy(result, pd.cache)
		pd.mu.RUnlock()
		return result, nil
	}
	pd.mu.RUnlock()

	pd.mu.Lock()
	defer pd.mu.Unlock()

	// Double-check after acquiring write lock (refinement #2)
	if !forceRefresh && time.Since(pd.lastRefresh) < pd.cacheTTL && pd.cache != nil {
		result := make([]connection.PrinterDetail, len(pd.cache))
		copy(result, pd.cache)
		return result, nil
	}

	printers, err := connection.ListAvailablePrinters()
	if err != nil {
		if pd.cache != nil {
			result := make([]connection.PrinterDetail, len(pd.cache))
			copy(result, pd.cache)
			return result, err // Return stale cache copy on error
		}
		return nil, err
	}

	pd.cache = printers
	pd.lastRefresh = time.Now()

	result := make([]connection.PrinterDetail, len(printers))
	copy(result, printers)
	return result, nil
}

// GetSummary returns a lightweight summary for health checks
func (pd *PrinterDiscovery) GetSummary() posprinter.Summary {
	printers, err := pd.GetPrinters(false)
	if err != nil {
		return posprinter.Summary{Status: "error", DetectedCount: 0}
	}

	thermal := connection.FilterThermalPrinters(printers)
	physical := connection.FilterPhysicalPrinters(printers)

	var defaultName string
	for _, p := range printers {
		if p.IsDefault {
			defaultName = p.Name
			break
		}
	}

	status := "ok"
	if len(thermal) == 0 && len(physical) > 0 {
		status = "warning"
	} else if len(physical) == 0 {
		status = "error"
	}

	return posprinter.Summary{
		Status:        status,
		DetectedCount: len(printers),
		ThermalCount:  len(thermal),
		DefaultName:   defaultName,
	}
}

// LogStartupDiagnostics logs printer info at service start
func (pd *PrinterDiscovery) LogStartupDiagnostics() {
	printers, err := pd.GetPrinters(true)
	if err != nil {
		log.Printf("[PRINTERS] ⚠️ Error enumerating printers:  %v", err)
		return
	}

	log.Println("[PRINTERS] ══════════════════════════════════════════════════")
	log.Printf("[PRINTERS] 🖨️ Detected %d installed printer(s)", len(printers))

	thermal := connection.FilterThermalPrinters(printers)
	if len(thermal) > 0 {
		log.Printf("[PRINTERS] 🎫 Thermal/POS printers:  %d", len(thermal))
		for _, p := range thermal {
			mark := ""
			if p.IsDefault {
				mark = " ⭐"
			}
			log.Printf("[PRINTERS]    • %s [%s] (%s)%s", p.Name, p.Port, p.Status, mark)
		}
	} else {
		log.Println("[PRINTERS] ⚠️ No thermal printers detected!")
	}

	if GetVerbose() {
		for _, p := range printers {
			if p.IsVirtual {
				log.Printf("[PRINTERS]    (virtual) %s", p.Name)
			}
		}
	}
	log.Println("[PRINTERS] ══════════════════════════════════════════════════")
}
