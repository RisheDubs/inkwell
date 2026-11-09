package httpapi

import (
	"errors"
	"net/http"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// getMe handles GET /api/me: the current user, created (with their journal) on first call.
func getMe(svc MeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		u, err := svc.Me(r.Context(), id.UserID, id.AccessToken)
		if err != nil {
			writeInternalError(w, r, "get me", err)
			return
		}
		writeJSON(w, http.StatusOK, u)
	}
}

// patchMe handles PATCH /api/me: update display name, handle or AI opt-in.
func patchMe(svc MeService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		var in journal.UpdateMeInput
		if err := decodeJSON(w, r, &in); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		u, err := svc.UpdateMe(r.Context(), id.UserID, in)
		var verr *journal.ValidationError
		switch {
		case err == nil:
			writeJSON(w, http.StatusOK, u)
		case errors.As(err, &verr):
			writeError(w, r, http.StatusBadRequest, "invalid_request", verr.Error())
		case errors.Is(err, journal.ErrHandleTaken):
			writeError(w, r, http.StatusConflict, "handle_taken", "that handle is already taken")
		case errors.Is(err, journal.ErrNotFound):
			writeError(w, r, http.StatusNotFound, "not_found", "user not found; call GET /api/me first")
		default:
			writeInternalError(w, r, "update me", err)
		}
	}
}
