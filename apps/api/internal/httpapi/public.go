package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// The public routes: anyone may read a journal its owner has made public, with no token.
// Every response is no-store, errors included: if the owner makes the journal private again,
// no browser or CDN may keep serving a copy.

// getPublicJournal handles GET /api/public/{handle}: title, theme and a page of entries.
func getPublicJournal(svc PublicService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		noStore(w) // before anything is written
		in, err := readListInput(r)
		if err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}
		j, err := svc.Journal(r.Context(), chi.URLParam(r, "handle"), in)
		if err != nil {
			writePublicError(w, r, "get public journal", err)
			return
		}
		writeJSON(w, http.StatusOK, j)
	}
}

// getPublicEntry handles GET /api/public/{handle}/entries/{id}: one entry with its body.
func getPublicEntry(svc PublicService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		e, err := svc.Entry(r.Context(), chi.URLParam(r, "handle"), chi.URLParam(r, "id"))
		if err != nil {
			writePublicError(w, r, "get public entry", err)
			return
		}
		writeJSON(w, http.StatusOK, e)
	}
}

// getPublicMediaURL handles GET /api/public/{handle}/media/{mediaID}: a short-lived image link.
func getPublicMediaURL(svc PublicService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		noStore(w)
		m, err := svc.MediaURL(r.Context(), chi.URLParam(r, "handle"), chi.URLParam(r, "mediaID"))
		if err != nil {
			writePublicError(w, r, "get public media url", err)
			return
		}
		writeJSON(w, http.StatusOK, m)
	}
}

// writePublicError gives one 404 for an unknown handle, a private journal, a malformed handle
// and an entry or image that isn't in the journal: the same code and message every time, so
// nobody can find out which handles exist or which journals are private.
func writePublicError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr *journal.ValidationError
	switch {
	case errors.As(err, &verr):
		writeError(w, r, http.StatusBadRequest, "invalid_request", verr.Error())
	case errors.Is(err, journal.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "not found")
	default:
		writeInternalError(w, r, op, err)
	}
}

// readListInput reads ?tag, ?cursor and ?limit the same way listEntries does.
func readListInput(r *http.Request) (journal.ListInput, error) {
	q := r.URL.Query()
	in := journal.ListInput{Tag: q.Get("tag"), Cursor: q.Get("cursor")}
	if s := q.Get("limit"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n < 1 {
			return journal.ListInput{}, errors.New("limit must be a whole number from 1 to 100")
		}
		in.Limit = n
	}
	return in, nil
}
