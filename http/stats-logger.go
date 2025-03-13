package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

type endpointStats struct {
	count       int
	totalTime   time.Duration
	lastPrinted time.Time
}

type statsLogger struct {
	stats         map[string]*endpointStats
	mu            sync.Mutex
	flushInterval time.Duration
}

func newStatsLogger() *statsLogger {
	sl := &statsLogger{
		stats:         make(map[string]*endpointStats),
		flushInterval: 5 * time.Second, // Print stats every 5 seconds
	}
	go sl.periodicFlush()
	return sl
}

func (sl *statsLogger) periodicFlush() {
	ticker := time.NewTicker(sl.flushInterval)
	for range ticker.C {
		sl.flushStats()
	}
}

func (sl *statsLogger) flushStats() {
	sl.mu.Lock()
	defer sl.mu.Unlock()

	now := time.Now()
	for endpoint, stats := range sl.stats {
		if stats.count > 0 && now.Sub(stats.lastPrinted) >= sl.flushInterval {
			// Convert to float64 and round to 2 decimal places
			avgTimeMs := float64(stats.totalTime.Microseconds()) / float64(stats.count) / 1000.0

			slog.Info("endpoint stats",
				"endpoint", endpoint,
				"count", stats.count,
				"avg_time_ms", fmt.Sprintf("%.2f", avgTimeMs),
				"period", sl.flushInterval,
			)
			// Reset stats after printing
			stats.count = 0
			stats.totalTime = 0
			stats.lastPrinted = now
		}
	}
}

func (sl *statsLogger) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		start := time.Now()

		next.ServeHTTP(w, r)

		duration := time.Since(start)

		sl.mu.Lock()
		if _, exists := sl.stats[endpoint]; !exists {
			sl.stats[endpoint] = &endpointStats{}
		}
		sl.stats[endpoint].count++
		sl.stats[endpoint].totalTime += duration
		sl.mu.Unlock()
	})
}
