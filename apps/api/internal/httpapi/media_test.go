package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// fakeMedia records what it was asked and returns canned links.
type fakeMedia struct {
	gotOwnerID     string
	gotContentType string
	gotMediaID     string
	err            error
}

func (f *fakeMedia) CreateUpload(_ context.Context, ownerID string, in journal.UploadRequest) (journal.Upload, error) {
	f.gotOwnerID, f.gotContentType = ownerID, in.ContentType
	if f.err != nil {
		return journal.Upload{}, f.err
	}
	return journal.Upload{
		MediaID:   "6730a1b2c3d4e5f601234567.png",
		URL:       "https://bucket.example/",
		Fields:    map[string]string{"key": "users/" + ownerID + "/media/6730a1b2c3d4e5f601234567.png"},
		MaxBytes:  journal.MaxUploadBytes,
		ExpiresAt: time.Date(2026, 11, 11, 9, 5, 0, 0, time.UTC),
	}, nil
}

func (f *fakeMedia) DownloadURL(_ context.Context, ownerID, mediaID string) (journal.MediaURL, error) {
	f.gotOwnerID, f.gotMediaID = ownerID, mediaID
	if f.err != nil {
		return journal.MediaURL{}, f.err
	}
	return journal.MediaURL{MediaID: mediaID, URL: "https://bucket.example/signed"}, nil
}

func TestMediaNeedsAuth(t *testing.T) {
	router := newTestRouter(t, testDeps{})
	for _, rt := range []struct{ method, path string }{
		{http.MethodPost, "/api/media/uploads"},
		{http.MethodGet, "/api/media/6730a1b2c3d4e5f601234567.png"},
	} {
		rec := do(t, router, rt.method, rt.path, "", `{"contentType":"image/png"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s status = %d, want %d", rt.method, rt.path, rec.Code, http.StatusUnauthorized)
		}
	}
}

func TestCreateUpload(t *testing.T) {
	media := &fakeMedia{}
	rec := do(t, newTestRouter(t, testDeps{media: media}), http.MethodPost, "/api/media/uploads", goodToken, `{"contentType":"image/png"}`)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body %s", rec.Code, http.StatusCreated, rec.Body)
	}
	if media.gotOwnerID != "user-123" || media.gotContentType != "image/png" {
		t.Errorf("service got owner %q type %q", media.gotOwnerID, media.gotContentType)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store (the response holds a signed link)", cc)
	}

	var up journal.Upload
	if err := json.NewDecoder(rec.Body).Decode(&up); err != nil {
		t.Fatal(err)
	}
	if up.MediaID == "" || up.URL == "" || up.Fields["key"] == "" || up.MaxBytes != journal.MaxUploadBytes {
		t.Errorf("upload = %+v", up)
	}
}

func TestCreateUploadErrors(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		err      error
		wantCode int
		wantErr  string
	}{
		{name: "client picks the key", body: `{"contentType":"image/png","key":"users/someone-else/media/x.png"}`, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "client picks the size limit", body: `{"contentType":"image/png","maxBytes":999999999}`, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "empty body", body: ``, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "not an image", body: `{"contentType":"text/html"}`, err: &journal.ValidationError{Field: "contentType", Message: "bad"}, wantCode: http.StatusBadRequest, wantErr: "invalid_request"},
		{name: "signing fails", body: `{"contentType":"image/png"}`, err: errors.New("no credentials for arn:aws:iam::123456789012:role/x"), wantCode: http.StatusInternalServerError, wantErr: "internal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := do(t, newTestRouter(t, testDeps{media: &fakeMedia{err: tt.err}}), http.MethodPost, "/api/media/uploads", goodToken, tt.body)
			if rec.Code != tt.wantCode {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tt.wantCode, rec.Body)
			}
			if code := errorCode(t, rec.Body.String()); code != tt.wantErr {
				t.Errorf("error code = %q, want %q", code, tt.wantErr)
			}
		})
	}
}

func TestGetMediaURL(t *testing.T) {
	media := &fakeMedia{}
	rec := do(t, newTestRouter(t, testDeps{media: media}), http.MethodGet, "/api/media/6730a1b2c3d4e5f601234567.png", goodToken, "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if media.gotOwnerID != "user-123" || media.gotMediaID != "6730a1b2c3d4e5f601234567.png" {
		t.Errorf("service got owner %q id %q", media.gotOwnerID, media.gotMediaID)
	}
	if cc := rec.Header().Get("Cache-Control"); cc != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cc)
	}
}

func TestGetMediaURLErrors(t *testing.T) {
	for _, tt := range []struct {
		err  error
		want int
	}{
		{journal.ErrNotFound, http.StatusNotFound},
		{errors.New("boom"), http.StatusInternalServerError},
	} {
		rec := do(t, newTestRouter(t, testDeps{media: &fakeMedia{err: tt.err}}), http.MethodGet, "/api/media/x.png", goodToken, "")
		if rec.Code != tt.want {
			t.Errorf("error %v: status = %d, want %d", tt.err, rec.Code, tt.want)
		}
	}
}
