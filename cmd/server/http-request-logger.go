package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/google/uuid"
	"github.com/programme-lv/backend/common/ctxlog"
)

const slowRequestThreshold = 500 * time.Millisecond

type httpReqInfo struct {
	method    string
	uri       string
	path      string
	referer   string
	ipaddr    string
	requestID string
	code      int
	written   int64
	duration  time.Duration
	userAgent string
	protocol  string
	tls       bool
}

type httpIPResolver struct {
	trustXForwardedFor bool
	trustXRealIP       bool
}

func (resolver *httpIPResolver) resolveIP(req *http.Request) string {
	ip, _, _ := net.SplitHostPort(req.RemoteAddr)

	if resolver.trustXRealIP {
		if realIP := req.Header.Get("X-Real-IP"); realIP != "" {
			ip = realIP
		}
	}

	if resolver.trustXForwardedFor {
		if forwardedFor := req.Header.Get("X-Forwarded-For"); forwardedFor != "" {
			ips := strings.Split(forwardedFor, ",")
			ip = strings.TrimSpace(ips[0])
		}
	}

	return ip
}

func logHTTPReq(ri *httpReqInfo) {
	attrs := []any{
		"req", ri.requestID,
		"m", ri.method,
		"uri", ri.uri,
		"status", ri.code,
		"kB", fmt.Sprintf("%d", ri.written/1000),
		"ms", ri.duration.Milliseconds(),
		"ip", ri.ipaddr,
		"ua", ri.userAgent,
	}

	switch {
	case ri.code >= 400:
		slog.Warn("http info", attrs...)
	case ri.path != "/subm-updates" && ri.duration >= slowRequestThreshold:
		// SSE /subm-updates duration is connection lifetime, not handler latency.
		slog.Info("http slow", attrs...)
	default:
		// Routine successful requests are metrics-only; avoid stdout spam.
	}

	slog.Debug("http info",
		"req-id", ri.requestID,
		"method", ri.method,
		"uri", ri.uri,
		"status", ri.code,
		"status-text", http.StatusText(ri.code),
		"written", fmt.Sprintf("%dB", ri.written),
		"duration", ri.duration,
		"ip", ri.ipaddr,
		"referer", ri.referer,
		"user-agent", ri.userAgent,
		"protocol", ri.protocol,
		"tls", ri.tls,
	)
}

func requestLoggerMiddleware(next http.Handler) http.Handler {
	ipResolver := &httpIPResolver{
		trustXForwardedFor: true,
		trustXRealIP:       true,
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := uuid.New().String()
		requestID = requestID[:8]

		w.Header().Set("X-Request-ID", requestID)

		reqInfo := &httpReqInfo{
			method:    r.Method,
			uri:       r.URL.String(),
			path:      r.URL.Path,
			referer:   r.Header.Get("Referer"),
			userAgent: r.Header.Get("User-Agent"),
			requestID: requestID,
			protocol:  r.Proto,
			tls:       r.TLS != nil,
		}

		reqInfo.ipaddr = ipResolver.resolveIP(r)

		reqLogger := slog.Default().With("req-id", requestID)

		ctx := ctxlog.WithLogger(r.Context(), reqLogger)
		r = r.WithContext(ctx)

		metrics := httpsnoop.CaptureMetrics(next, w, r)

		reqInfo.code = metrics.Code
		reqInfo.written = metrics.Written
		reqInfo.duration = metrics.Duration

		recordHTTPMetrics(r, reqInfo.method, reqInfo.code, reqInfo.written, reqInfo.duration)
		logHTTPReq(reqInfo)
	})
}
