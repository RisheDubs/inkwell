package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// identity is who made the request, taken from their verified access token.
type identity struct {
	UserID      string
	AccessToken string
}

type identityKey struct{}

// identityFrom returns the identity requireAuth stored in the request context.
func identityFrom(ctx context.Context) (identity, bool) {
	id, ok := ctx.Value(identityKey{}).(identity)
	return id, ok && id.UserID != ""
}

// requireAuth rejects requests without a valid access token. The user ID always comes
// from the verified token, never from anything else the client sends.
func requireAuth(v TokenVerifier) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token, ok := bearerToken(r.Header.Get("Authorization"))
			if !ok {
				unauthorized(w, r, "missing or malformed Authorization header; expected: Bearer <access token>")
				return
			}

			claims, err := v.Verify(token)
			if err != nil {
				// Log why (useful for debugging); never log the token itself.
				slog.Warn("auth: token rejected", "err", err, "request_id", middleware.GetReqID(r.Context()))
				unauthorized(w, r, "invalid or expired access token")
				return
			}

			ctx := context.WithValue(r.Context(), identityKey{}, identity{UserID: claims.Subject, AccessToken: token})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// bearerToken extracts the token from "Authorization: Bearer <token>".
func bearerToken(header string) (string, bool) {
	scheme, token, found := strings.Cut(header, " ")
	if !found || !strings.EqualFold(scheme, "Bearer") {
		return "", false
	}
	token = strings.TrimSpace(token)
	return token, token != ""
}

func unauthorized(w http.ResponseWriter, r *http.Request, message string) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="inkwell"`)
	writeError(w, r, http.StatusUnauthorized, "unauthorized", message)
}

// exposeRequestID returns the request ID to the client, so they can quote it in bug reports.
func exposeRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id := middleware.GetReqID(r.Context()); id != "" {
			w.Header().Set("X-Request-Id", id)
		}
		next.ServeHTTP(w, r)
	})
}

// requestLogger writes one structured log line per request. It logs the path but not the
// query string or headers, which could contain secrets.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		status := ww.Status()
		if status == 0 {
			status = http.StatusOK
		}
		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"request_id", middleware.GetReqID(r.Context()),
		)
	})
}
