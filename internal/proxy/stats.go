package proxy

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// StatsTracker tracks server statistics for the landing page API
type StatsTracker struct {
	startTime      time.Time
	totalRelays    atomic.Int64
	totalBytes     atomic.Int64
	totalErrors    atomic.Int64
	
	// Rolling window for throughput calculation (bytes per second)
	bytesWindow    []int64
	bytesWindowMu  sync.Mutex
	
	// History for 24h chart (hourly samples)
	history        []HistorySample
	historyMu      sync.RWMutex
	AllowedOrigin  string
}

// HistorySample represents a single data point for historical charts
type HistorySample struct {
	Time    string `json:"time"`
	Users   int64  `json:"users"`
	Traffic int64  `json:"traffic"`
}

// StatsResponse is the JSON response for /api/stats
type StatsResponse struct {
	TotalUsers        int64   `json:"totalUsers"`
	ActiveConnections int     `json:"activeConnections"`
	UptimeSeconds     int64   `json:"uptimeSeconds"`
	DataThroughput    string  `json:"dataThroughput"`
	Latency           int     `json:"latency"`
	SuccessRate       float64 `json:"successRate"`
}

// Global stats tracker instance
var Stats = &StatsTracker{
	startTime:   time.Now(),
	bytesWindow: make([]int64, 0, 60),
	history:     make([]HistorySample, 0, 24),
}

func init() {
	// Start background goroutine to update rolling window and history
	go Stats.backgroundUpdater()
}

// backgroundUpdater runs every second to update rolling metrics
func (s *StatsTracker) backgroundUpdater() {
	secondTicker := time.NewTicker(1 * time.Second)
	hourTicker := time.NewTicker(1 * time.Hour)
	defer secondTicker.Stop()
	defer hourTicker.Stop()

	var lastBytes int64

	for {
		select {
		case <-secondTicker.C:
			// Update bytes per second rolling window
			currentBytes := s.totalBytes.Load()
			bytesThisSecond := currentBytes - lastBytes
			lastBytes = currentBytes

			s.bytesWindowMu.Lock()
			s.bytesWindow = append(s.bytesWindow, bytesThisSecond)
			if len(s.bytesWindow) > 60 {
				s.bytesWindow = s.bytesWindow[1:]
			}
			s.bytesWindowMu.Unlock()

		case <-hourTicker.C:
			// Record hourly sample for history
			s.historyMu.Lock()
			s.history = append(s.history, HistorySample{
				Time:    time.Now().Format("15:04"),
				Users:   s.totalRelays.Load(),
				Traffic: s.totalBytes.Load(),
			})
			if len(s.history) > 24 {
				s.history = s.history[1:]
			}
			s.historyMu.Unlock()
		}
	}
}

// RecordRelay records a successful relay connection
func (s *StatsTracker) RecordRelay() {
	s.totalRelays.Add(1)
}

// RecordBytes records bytes transferred
func (s *StatsTracker) RecordBytes(n int64) {
	s.totalBytes.Add(n)
}

// RecordError records an error
func (s *StatsTracker) RecordError() {
	s.totalErrors.Add(1)
}

// GetThroughput calculates average bytes per second over the last minute
func (s *StatsTracker) GetThroughput() string {
	s.bytesWindowMu.Lock()
	defer s.bytesWindowMu.Unlock()

	if len(s.bytesWindow) == 0 {
		return "0 B/s"
	}

	var total int64
	for _, b := range s.bytesWindow {
		total += b
	}
	avg := total / int64(len(s.bytesWindow))

	return formatBytes(avg) + "/s"
}

// GetSuccessRate calculates the success rate percentage
func (s *StatsTracker) GetSuccessRate() float64 {
	relays := s.totalRelays.Load()
	errors := s.totalErrors.Load()
	
	total := relays + errors
	if total == 0 {
		return 100.0
	}
	
	return float64(relays) / float64(total) * 100.0
}

// GetStats returns the current stats for the API
func (s *StatsTracker) GetStats() StatsResponse {
	return StatsResponse{
		TotalUsers:        s.totalRelays.Load(),
		ActiveConnections: GetActiveConns(),
		UptimeSeconds:     int64(time.Since(s.startTime).Seconds()),
		DataThroughput:    s.GetThroughput(),
		Latency:           18, // TODO: Implement actual latency tracking
		SuccessRate:       s.GetSuccessRate(),
	}
}

// GetHistory returns historical data for charts
func (s *StatsTracker) GetHistory() []HistorySample {
	s.historyMu.RLock()
	defer s.historyMu.RUnlock()

	// Return a copy to avoid race conditions
	result := make([]HistorySample, len(s.history))
	copy(result, s.history)
	return result
}

// formatBytes converts bytes to human readable format
func formatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return formatFloat(float64(bytes)/float64(GB)) + " GB"
	case bytes >= MB:
		return formatFloat(float64(bytes)/float64(MB)) + " MB"
	case bytes >= KB:
		return formatFloat(float64(bytes)/float64(KB)) + " KB"
	default:
		return itoa(int(bytes)) + " B"
	}
}

// formatFloat formats a float with 1 decimal place
func formatFloat(f float64) string {
	intPart := int(f)
	decPart := int((f - float64(intPart)) * 10)
	if decPart < 0 {
		decPart = -decPart
	}
	return itoa(intPart) + "." + itoa(decPart)
}

// StatsHandler handles /api/stats requests
func StatsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", Stats.AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	stats := Stats.GetStats()
	json.NewEncoder(w).Encode(stats)
}

// HistoryHandler handles /api/history requests
func HistoryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", Stats.AllowedOrigin)
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	history := Stats.GetHistory()
	
	// If no history yet, generate mock data for the chart
	if len(history) == 0 {
		history = generateInitialHistory()
	}
	
	json.NewEncoder(w).Encode(history)
}

// generateInitialHistory creates initial history data for new deployments
func generateInitialHistory() []HistorySample {
	now := time.Now()
	history := make([]HistorySample, 24)
	
	for i := 0; i < 24; i++ {
		t := now.Add(time.Duration(i-23) * time.Hour)
		history[i] = HistorySample{
			Time:    t.Format("15:04"),
			Users:   0,
			Traffic: 0,
		}
	}
	
	return history
}
