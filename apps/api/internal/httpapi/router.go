package httpapi

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
)

// NewRouter builds the HTTP router with every route registered.
func NewRouter() http.Handler {
	r := chi.NewRouter()

	r.Get("/healthz", healthz)
	return r
}

// healthz reports that the API process is alive.
func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]string{"status": "ok"}); err != nil {
		slog.Error("healthz: write response", "err", err)
	}
}
