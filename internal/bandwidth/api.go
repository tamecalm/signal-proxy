package bandwidth

import (
	"encoding/json"
	"net/http"
)

// UsageEntry represents a single user's bandwidth usage for the API
type UsageEntry struct {
	BytesUp        int64   `json:"bytes_up"`
	BytesDown      int64   `json:"bytes_down"`
	TotalGB        float64 `json:"total_gb"`
	LimitGB        int     `json:"limit_gb"`
	PercentUsed    float64 `json:"percent_used"`
	ActiveConns    int     `json:"active_conns"`
}

// UsageResponse is the JSON response for /api/usage
type UsageResponse struct {
	Month string                `json:"month"`
	Users map[string]UsageEntry `json:"users"`
}

// UsageHandler returns an http.HandlerFunc for the /api/usage endpoint.
// It needs a reference to the tracker and an allowed origin for CORS.
func UsageHandler(tracker *Tracker, allowedOrigin string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		allUsage := tracker.GetAllUsage()

		resp := UsageResponse{
			Month: tracker.GetMonth(),
			Users: make(map[string]UsageEntry, len(allUsage)),
		}

		for username, usage := range allUsage {
			totalGB := float64(usage.TotalBytes) / (1024 * 1024 * 1024)
			resp.Users[username] = UsageEntry{
				BytesUp:     usage.BytesUp,
				BytesDown:   usage.BytesDown,
				TotalGB:     totalGB,
				LimitGB:     0, // Will be enriched by caller if needed
				PercentUsed: 0,
				ActiveConns: usage.ActiveConns,
			}
		}

		json.NewEncoder(w).Encode(resp)
	}
}
