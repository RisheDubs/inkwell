package httpapi

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// createUpload handles POST /api/media/uploads: a 5-minute link to upload one image to S3.
func createUpload(svc MediaService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		var in journal.UploadRequest
		if err := decodeJSON(w, r, &in); err != nil {
			writeError(w, r, http.StatusBadRequest, "invalid_request", err.Error())
			return
		}

		up, err := svc.CreateUpload(r.Context(), id.UserID, in)
		var verr *journal.ValidationError
		switch {
		case err == nil:
			noStore(w)
			writeJSON(w, http.StatusCreated, up)
		case errors.As(err, &verr):
			writeError(w, r, http.StatusBadRequest, "invalid_request", verr.Error())
		default:
			writeInternalError(w, r, "create upload", err)
		}
	}
}

// getMediaURL handles GET /api/media/{mediaID}: a 15-minute link to read one of the caller's images.
func getMediaURL(svc MediaService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := identityFrom(r.Context())
		if !ok {
			unauthorized(w, r, "not signed in")
			return
		}

		m, err := svc.DownloadURL(r.Context(), id.UserID, chi.URLParam(r, "mediaID"))
		switch {
		case err == nil:
			noStore(w)
			writeJSON(w, http.StatusOK, m)
		case errors.Is(err, journal.ErrNotFound):
			writeError(w, r, http.StatusNotFound, "not_found", "media not found")
		default:
			writeInternalError(w, r, "get media url", err)
		}
	}
}

// noStore stops browsers and proxies from caching a response that contains a signed link.
func noStore(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "no-store")
}
