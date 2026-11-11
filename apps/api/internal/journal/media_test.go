package journal

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakePresigner records what it was asked to sign and returns predictable links.
type fakePresigner struct {
	key         string
	contentType string
	maxBytes    int64
	expires     time.Duration
	err         error
}

func (f *fakePresigner) PresignUpload(_ context.Context, key, contentType string, maxBytes int64, expires time.Duration) (string, map[string]string, error) {
	f.key, f.contentType, f.maxBytes, f.expires = key, contentType, maxBytes, expires
	if f.err != nil {
		return "", nil, f.err
	}
	return "https://bucket.example/", map[string]string{"key": key, "Content-Type": contentType}, nil
}

func (f *fakePresigner) PresignDownload(_ context.Context, key string, expires time.Duration) (string, error) {
	f.key, f.expires = key, expires
	if f.err != nil {
		return "", f.err
	}
	return "https://bucket.example/" + key + "?signature", nil
}

func newTestMediaService(p *fakePresigner) *MediaService {
	s := NewMediaService(p, func() string { return "6730a1b2c3d4e5f601234567" })
	s.now = func() time.Time { return entryTestNow }
	return s
}

func TestCreateUpload(t *testing.T) {
	p := &fakePresigner{}
	got, err := newTestMediaService(p).CreateUpload(context.Background(), "user-1", UploadRequest{ContentType: " Image/PNG "})
	if err != nil {
		t.Fatalf("CreateUpload() error = %v", err)
	}

	wantID := "6730a1b2c3d4e5f601234567.png"
	if got.MediaID != wantID {
		t.Errorf("MediaID = %q, want %q", got.MediaID, wantID)
	}
	if p.key != "users/user-1/media/"+wantID {
		t.Errorf("signed key = %q, want it inside users/user-1/media/", p.key)
	}
	if p.contentType != "image/png" {
		t.Errorf("signed content type = %q, want image/png (normalised)", p.contentType)
	}
	if p.maxBytes != MaxUploadBytes || got.MaxBytes != MaxUploadBytes {
		t.Errorf("max bytes = %d / %d, want %d", p.maxBytes, got.MaxBytes, MaxUploadBytes)
	}
	if p.expires != 5*time.Minute {
		t.Errorf("expiry = %v, want 5m", p.expires)
	}
	if !got.ExpiresAt.Equal(entryTestNow.Add(5 * time.Minute)) {
		t.Errorf("ExpiresAt = %v, want now + 5m", got.ExpiresAt)
	}
	if got.Fields["Content-Type"] != "image/png" {
		t.Errorf("fields = %v, want the Content-Type the client must send", got.Fields)
	}
}

func TestCreateUploadRejectsNonImages(t *testing.T) {
	for _, ct := range []string{"", "text/html", "image/svg+xml", "application/pdf", "image/png; charset=utf-8"} {
		p := &fakePresigner{}
		_, err := newTestMediaService(p).CreateUpload(context.Background(), "user-1", UploadRequest{ContentType: ct})
		var verr *ValidationError
		if !errors.As(err, &verr) || verr.Field != "contentType" {
			t.Errorf("CreateUpload(%q) error = %v, want a ValidationError on contentType", ct, err)
		}
		if p.key != "" {
			t.Errorf("CreateUpload(%q) signed %q; nothing should be signed for a rejected type", ct, p.key)
		}
	}
}

func TestDownloadURLStaysInCallersFolder(t *testing.T) {
	p := &fakePresigner{}
	got, err := newTestMediaService(p).DownloadURL(context.Background(), "user-b", "6730a1b2c3d4e5f601234567.jpg")
	if err != nil {
		t.Fatalf("DownloadURL() error = %v", err)
	}
	// Even if user-b guessed user-a's media ID, the key is built from user-b's own ID.
	if p.key != "users/user-b/media/6730a1b2c3d4e5f601234567.jpg" {
		t.Errorf("signed key = %q, want it inside users/user-b/media/", p.key)
	}
	if p.expires != 15*time.Minute || !got.ExpiresAt.Equal(entryTestNow.Add(15*time.Minute)) {
		t.Errorf("expiry = %v / %v, want 15m", p.expires, got.ExpiresAt)
	}
}

func TestDownloadURLRejectsMalformedIDs(t *testing.T) {
	for _, id := range []string{
		"",
		"6730a1b2c3d4e5f601234567",     // no extension
		"6730a1b2c3d4e5f601234567.svg", // not an allowed type
		"../user-a/media/6730a1b2c3d4e5f601234567.png", // path traversal
		"6730A1B2C3D4E5F601234567.png",                 // uppercase hex
		"6730a1b2c3d4e5f601234567.png/../x",
	} {
		p := &fakePresigner{}
		if _, err := newTestMediaService(p).DownloadURL(context.Background(), "user-b", id); !errors.Is(err, ErrNotFound) {
			t.Errorf("DownloadURL(%q) error = %v, want ErrNotFound", id, err)
		}
		if p.key != "" {
			t.Errorf("DownloadURL(%q) signed %q; nothing should be signed for a bad ID", id, p.key)
		}
	}
}

func TestMediaPresignErrorsAreWrapped(t *testing.T) {
	cause := errors.New("no credentials")
	s := newTestMediaService(&fakePresigner{err: cause})

	if _, err := s.CreateUpload(context.Background(), "user-1", UploadRequest{ContentType: "image/jpeg"}); !errors.Is(err, cause) {
		t.Errorf("CreateUpload() error = %v, want it to wrap %v", err, cause)
	}
	if _, err := s.DownloadURL(context.Background(), "user-1", "6730a1b2c3d4e5f601234567.jpg"); !errors.Is(err, cause) {
		t.Errorf("DownloadURL() error = %v, want it to wrap %v", err, cause)
	}
}
