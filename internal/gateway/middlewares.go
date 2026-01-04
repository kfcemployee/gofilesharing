package gateway

import (
	"log/slog"
	"net/http"
)

func loggingMiddleware(lg *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		lg.Info("New Request.",
			slog.String("Method", r.Method),
			slog.String("Path", r.URL.Path),
		)

		next.ServeHTTP(w, r)
	})
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // Разрешить всем
		next.ServeHTTP(w, r)
	})
}
