package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// An entry request carries a Tiptap body of up to journal.MaxBodyBytes, plus a little metadata.
const maxEntryRequestBytes = journal.MaxBodyBytes + 8<<10

// listEntries handles GET /api/entries: one page of the caller's entries, newest first.
func listEntries(svc EntryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		q := r.URL.Query()
		in := journal.ListInput{Tag: q.Get("tag"), Cursor: q.Get("cursor")}
		if s := q.Get("limit"); s != "" {
			n, err := strconv.Atoi(s)
			if err != nil || n < 1 {
				writeError(w, r, http.StatusBadRequest, "invalid_request", "limit must be a whole number from 1 to 100")
				return
			}
			in.Limit = n
		}

		page, err := svc.ListEntries(r.Context(), id.UserID, in)
		if err != nil {
			writeEntryError(w, r, "list entries", err)
			return
		}
		writeJSON(w, http.StatusOK, page)
	}
}

// createEntry handles POST /api/entries.
func createEntry(svc EntryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		var in journal.CreateEntryInput
		if err := decodeJSONLimit(w, r, &in, maxEntryRequestBytes); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		e, err := svc.CreateEntry(r.Context(), id.UserID, in)
		if errors.Is(err, journal.ErrNotFound) {
			writeError(w, r, http.StatusNotFound, "not_found", "journal not found; call GET /api/me first")
			return
		}
		if err != nil {
			writeEntryError(w, r, "create entry", err)
			return
		}
		w.Header().Set("Location", "/api/entries/"+e.ID)
		writeJSON(w, http.StatusCreated, e)
	}
}

// getEntry handles GET /api/entries/{id}: one entry with its body.
func getEntry(svc EntryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		e, err := svc.Entry(r.Context(), id.UserID, chi.URLParam(r, "id"))
		if err != nil {
			writeEntryError(w, r, "get entry", err)
			return
		}
		writeJSON(w, http.StatusOK, e)
	}
}

// patchEntry handles PATCH /api/entries/{id}.
func patchEntry(svc EntryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		var in journal.UpdateEntryInput
		if err := decodeJSONLimit(w, r, &in, maxEntryRequestBytes); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		e, err := svc.UpdateEntry(r.Context(), id.UserID, chi.URLParam(r, "id"), in)
		if err != nil {
			writeEntryError(w, r, "update entry", err)
			return
		}
		writeJSON(w, http.StatusOK, e)
	}
}

// deleteEntry handles DELETE /api/entries/{id}.
func deleteEntry(svc EntryService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		if err := svc.DeleteEntry(r.Context(), id.UserID, chi.URLParam(r, "id")); err != nil {
			writeEntryError(w, r, "delete entry", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// writeEntryError maps entry errors to responses. Someone else's entry is a 404, never a 403:
// a 403 would confirm that the ID exists.
func writeEntryError(w http.ResponseWriter, r *http.Request, op string, err error) {
	var verr *journal.ValidationError
	switch {
	case errors.As(err, &verr):
		writeError(w, r, http.StatusBadRequest, "invalid_request", verr.Error())
	case errors.Is(err, journal.ErrNotFound):
		writeError(w, r, http.StatusNotFound, "not_found", "entry not found")
	default:
		writeInternalError(w, r, op, err)
	}
}
