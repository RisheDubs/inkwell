package journal

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

const (
	// MaxUploadBytes caps one image upload at 5 MB. S3 enforces it, not the API: the
	// browser sends the file straight to S3, and the signed policy refuses anything bigger.
	MaxUploadBytes = 5 << 20

	uploadExpiry   = 5 * time.Minute  // how long a client has to start its upload
	downloadExpiry = 15 * time.Minute // how long an image link works once handed out
)

// uploadTypes is the allow-list of image types, mapped to the file extension we store them under.
var uploadTypes = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/webp": "webp",
	"image/gif":  "gif",
}

// A media ID is a server-made ID plus the extension of an allowed type, e.g. "6730a1b2c3d4e5f601234567.png".
var mediaIDPattern = regexp.MustCompile(`^[0-9a-f]{24}\.(jpg|png|webp|gif)$`)

// UploadRequest is what POST /api/media/uploads accepts.
type UploadRequest struct {
	ContentType string `json:"contentType"`
}

// Upload tells the client how to send one image straight to S3: an HTML form POST to URL
// with every entry in Fields as a form field, and the file as the last field, named "file".
type Upload struct {
	MediaID   string            `json:"mediaId"`
	URL       string            `json:"url"`
	Fields    map[string]string `json:"fields"`
	MaxBytes  int64             `json:"maxBytes"`
	ExpiresAt time.Time         `json:"expiresAt"`
}

// MediaURL is a short-lived link for reading one of the caller's images.
type MediaURL struct {
	MediaID   string    `json:"mediaId"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expiresAt"`
}

// Presigner creates short-lived S3 links. It never sees the file itself.
type Presigner interface {
	// PresignUpload returns a form-POST URL and fields that only accept an object at key,
	// with exactly contentType, no larger than maxBytes, within expires.
	PresignUpload(ctx context.Context, key, contentType string, maxBytes int64, expires time.Duration) (url string, fields map[string]string, err error)
	// PresignDownload returns a GET URL for key that works for expires.
	PresignDownload(ctx context.Context, key string, expires time.Duration) (string, error)
}

// MediaKey is where an image lives: users/{ownerID}/media/{mediaID}.
func MediaKey(ownerID, mediaID string) string {
	return "users/" + ownerID + "/media/" + mediaID
}

// MediaService hands out upload and download links for the caller's images.
// It stores nothing itself: every key is built from the caller's user ID, so a link
// can only ever point inside the caller's own folder.
type MediaService struct {
	presigner Presigner
	newID     func() string
	now       func() time.Time
}

// NewMediaService creates a MediaService. newID makes the random part of each media ID.
func NewMediaService(presigner Presigner, newID func() string) *MediaService {
	return &MediaService{presigner: presigner, newID: newID, now: time.Now}
}

// CreateUpload checks the requested type and returns a 5-minute upload link for a new image.
func (s *MediaService) CreateUpload(ctx context.Context, ownerID string, in UploadRequest) (Upload, error) {
	contentType := strings.ToLower(strings.TrimSpace(in.ContentType))
	ext, ok := uploadTypes[contentType]
	if !ok {
		return Upload{}, &ValidationError{Field: "contentType", Message: "must be one of image/jpeg, image/png, image/webp or image/gif"}
	}

	mediaID := s.newID() + "." + ext
	url, fields, err := s.presigner.PresignUpload(ctx, MediaKey(ownerID, mediaID), contentType, MaxUploadBytes, uploadExpiry)
	if err != nil {
		return Upload{}, fmt.Errorf("presign upload: %w", err)
	}
	return Upload{
		MediaID:   mediaID,
		URL:       url,
		Fields:    fields,
		MaxBytes:  MaxUploadBytes,
		ExpiresAt: storedTime(s.now()).Add(uploadExpiry),
	}, nil
}

// DownloadURL returns a 15-minute link to one of the caller's images. A malformed ID is
// ErrNotFound, exactly like a missing one.
func (s *MediaService) DownloadURL(ctx context.Context, ownerID, mediaID string) (MediaURL, error) {
	if !mediaIDPattern.MatchString(mediaID) {
		return MediaURL{}, ErrNotFound
	}
	url, err := s.presigner.PresignDownload(ctx, MediaKey(ownerID, mediaID), downloadExpiry)
	if err != nil {
		return MediaURL{}, fmt.Errorf("presign download: %w", err)
	}
	return MediaURL{MediaID: mediaID, URL: url, ExpiresAt: storedTime(s.now()).Add(downloadExpiry)}, nil
}
