package httpapi

import (
	"errors"
	"net/http"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// getJournal handles GET /api/journal: the signed-in user's journal and theme.
func getJournal(svc JournalService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		j, err := svc.Journal(r.Context(), id.UserID)
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, j)
		case errors.Is(err, journal.ErrNotFound):
			writeError(w, r, http.StatusNotFound, "not_found", "journal not found; call GET /api/me first")
		default:
			writeInternalError(w, r, "get journal", err)
		}
	}
}

// patchJournal handles PATCH /api/journal: title, visibility and theme.
func patchJournal(svc JournalService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		var in journal.UpdateJournalInput
		if err := decodeJSON(w, r, &in); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		j, err := svc.UpdateJournal(r.Context(), id.UserID, in)
		var verr *journal.ValidationError
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, j)
		case errors.As(err, &verr):
			writeError(w, r, http.StatusBadRequest, "invalid_request", verr.Error())
		case errors.Is(err, journal.ErrNotFound):
			writeError(w, r, http.StatusNotFound, "not_found", "journal not found; call GET /api/me first")
		default:
			writeInternalError(w, r, "update journal", err)
		}
	}
}
