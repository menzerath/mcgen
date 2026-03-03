package web

import (
	"log/slog"
	"net/http"
	"time"

	middleware "github.com/go-chi/chi/v5/middleware"
)

// slogLoggingMiddleware logs each completed HTTP request using slog.
// 5xx → Error, 4xx → Warn, everything else → Debug.
func slogLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		start := time.Now()
		next.ServeHTTP(ww, r)
		duration := time.Since(start)

		status := ww.Status()
		level := slog.LevelDebug
		if status >= 500 {
			level = slog.LevelError
		} else if status >= 400 {
			level = slog.LevelWarn
		}

		slog.Log(r.Context(), level, "http request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration", duration,
			"remote", r.RemoteAddr,
		)
	})
}
