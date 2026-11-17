// Package httpapi exposes Inkwell over HTTP.
package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/RisheDubs/inkwell/apps/api/internal/auth"
	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// Pinger reports whether a dependency, such as the database, is reachable.
type Pinger interface {
	Ping(ctx context.Context) error
}

// TokenVerifier checks an access token and returns its claims.
type TokenVerifier interface {
	Verify(token string) (auth.Claims, error)
}

// MeService handles the signed-in user's own profile.
type MeService interface {
	Me(ctx context.Context, userID, accessToken string) (journal.User, error)
	UpdateMe(ctx context.Context, userID string, in journal.UpdateMeInput) (journal.User, error)
}

// JournalService handles the signed-in user's journal.
type JournalService interface {
	Journal(ctx context.Context, ownerID string) (journal.Journal, error)
	UpdateJournal(ctx context.Context, ownerID string, in journal.UpdateJournalInput) (journal.Journal, error)
}

// EntryService handles the signed-in user's entries.
type EntryService interface {
	CreateEntry(ctx context.Context, ownerID string, in journal.CreateEntryInput) (journal.EntryWithBody, error)
	Entry(ctx context.Context, ownerID, id string) (journal.EntryWithBody, error)
	ListEntries(ctx context.Context, ownerID string, in journal.ListInput) (journal.EntryPage, error)
	UpdateEntry(ctx context.Context, ownerID, id string, in journal.UpdateEntryInput) (journal.EntryWithBody, error)
	DeleteEntry(ctx context.Context, ownerID, id string) error
}

// MediaService hands out short-lived upload and download links for the signed-in user's images.
type MediaService interface {
	CreateUpload(ctx context.Context, ownerID string, in journal.UploadRequest) (journal.Upload, error)
	DownloadURL(ctx context.Context, ownerID, mediaID string) (journal.MediaURL, error)
}

// PublicService serves journals their owners have made public, to anyone, by handle.
type PublicService interface {
	Journal(ctx context.Context, handle string, in journal.ListInput) (journal.PublicJournal, error)
	Entry(ctx context.Context, handle, id string) (journal.PublicEntry, error)
	MediaURL(ctx context.Context, handle, mediaID string) (journal.MediaURL, error)
}

// Deps are everything the router needs. Tests pass fakes.
type Deps struct {
	DB       Pinger
	Verifier TokenVerifier
	Me       MeService
	Journals JournalService
	Entries  EntryService
	Media    MediaService
	Public   PublicService
}

// NewRouter builds the HTTP router with every route registered.
func NewRouter(d Deps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID) // give every request an ID
	r.Use(exposeRequestID)      // and send it back in X-Request-Id
	r.Use(requestLogger)        // one JSON log line per request
	r.Use(middleware.Recoverer) // a panic becomes a 500, not a crashed server

	// Public: no token needed.
	r.Get("/healthz", healthz(d.DB))

	r.Route("/api", func(r chi.Router) {
		// Public journals: read-only, no token.
		r.Route("/public/{handle}", func(r chi.Router) {
			// Its own 404 and 405, set before chi mounts it. Without them it would inherit the
			// token-checking ones below, and a mistyped public link would ask for a token.
			r.NotFound(publicNotFound)
			r.MethodNotAllowed(methodNotAllowed)

			r.Get("/", getPublicJournal(d.Public))
			r.Get("/entries/{id}", getPublicEntry(d.Public))
			r.Get("/media/{mediaID}", getPublicMediaURL(d.Public))
		})

		// Everything else under /api needs a valid Cognito access token.
		r.Group(func(r chi.Router) {
			r.Use(requireAuth(d.Verifier))

			r.Get("/me", getMe(d.Me))
			r.Patch("/me", patchMe(d.Me))

			r.Get("/journal", getJournal(d.Journals))
			r.Patch("/journal", patchJournal(d.Journals))

			r.Get("/entries", listEntries(d.Entries))
			r.Post("/entries", createEntry(d.Entries))
			r.Get("/entries/{id}", getEntry(d.Entries))
			r.Patch("/entries/{id}", patchEntry(d.Entries))
			r.Delete("/entries/{id}", deleteEntry(d.Entries))

			r.Post("/media/uploads", createUpload(d.Media))
			r.Get("/media/{mediaID}", getMediaURL(d.Media))
		})

		// Unknown paths and wrong methods under /api check the token first too, so a caller
		// without one can't work out which private routes exist.
		authed := requireAuth(d.Verifier)
		r.NotFound(authed(http.HandlerFunc(notFound)).ServeHTTP)
		r.MethodNotAllowed(authed(http.HandlerFunc(methodNotAllowed)).ServeHTTP)
	})

	return r
}

// notFound is the API's 404, in the standard error format.
func notFound(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusNotFound, "not_found", "not found")
}

// publicNotFound is the same 404 for a path that doesn't exist under /api/public/{handle}.
func publicNotFound(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	notFound(w, r)
}

// methodNotAllowed is the API's 405, in the standard error format.
func methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

// healthz reports whether the API is alive and can reach MongoDB.
func healthz(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			logError(r, "healthz: database ping failed", err)
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "unavailable"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
