package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// maxBodyBytes caps request bodies. A profile update is tiny; anything bigger is rejected.
const maxBodyBytes = 4 << 10 // 4 KB

// errorResponse is the API's single error format:
// {"error": {"code": "...", "message": "...", "requestId": "..."}}
type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
}

// writeJSON sends v as a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write JSON response", "err", err)
	}
}

// writeError sends an error in the standard format. message is shown to the client,
// so it must never contain internal details.
func writeError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorDetail{
		Code:      code,
		Message:   message,
		RequestID: middleware.GetReqID(r.Context()),
	}})
}

// writeInternalError logs the real cause and tells the client only that something failed.
func writeInternalError(w http.ResponseWriter, r *http.Request, msg string, err error) {
	logError(r, msg, err)
	writeError(w, r, http.StatusInternalServerError, "internal", "something went wrong; quote the requestId if you report it")
}

func logError(r *http.Request, msg string, err error) {
	slog.Error(msg, "err", err, "request_id", middleware.GetReqID(r.Context()))
}

// decodeJSON reads exactly one JSON object into dst, rejecting unknown fields and oversized bodies.
// The returned error's message is safe to show the client.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(dst); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			return fmt.Errorf("request body must be %d bytes or smaller", maxBodyBytes)
		case errors.Is(err, io.EOF):
			return errors.New("request body must not be empty")
		default:
			return fmt.Errorf("request body is not valid: %v", err)
		}
	}
	// Anything after the first object (e.g. `{}{}`) is rejected too.
	if err := dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request body must contain a single JSON object")
	}
	return nil
}
