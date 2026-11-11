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

// Deps are everything the router needs. Tests pass fakes.
type Deps struct {
	DB       Pinger
	Verifier TokenVerifier
	Me       MeService
	Journals JournalService
	Entries  EntryService
	Media    MediaService
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

	// Everything under /api needs a valid Cognito access token.
	r.Route("/api", func(r chi.Router) {
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

	return r
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
