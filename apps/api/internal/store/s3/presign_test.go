package s3

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// realPresigner signs with fake, static credentials. Presigning is pure computation, so these
// tests exercise the real AWS SDK without any network access or AWS account.
func realPresigner() *Presigner {
	client := awss3.New(awss3.Options{
		Region:      "ap-southeast-2",
		Credentials: credentials.NewStaticCredentialsProvider("test-access-key-id", "test-secret", ""),
	})
	return NewPresigner(awss3.NewPresignClient(client), testBucket)
}

// decodePolicy turns the base64 "policy" field back into its conditions.
func decodePolicy(t *testing.T, fields map[string]string) []any {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(fields["policy"])
	if err != nil {
		t.Fatalf("policy is not base64: %v", err)
	}
	var doc struct {
		Expiration string `json:"expiration"`
		Conditions []any  `json:"conditions"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("policy is not JSON: %v", err)
	}
	return doc.Conditions
}

func TestPresignUploadPolicy(t *testing.T) {
	key := "users/user-1/media/6730a1b2c3d4e5f601234567.png"
	u, fields, err := realPresigner().PresignUpload(context.Background(), key, "image/png", 5<<20, 5*time.Minute)
	if err != nil {
		t.Fatalf("PresignUpload() error = %v", err)
	}

	if !strings.HasPrefix(u, "https://") || !strings.Contains(u, testBucket) {
		t.Errorf("URL = %q, want an https URL for the bucket", u)
	}
	if fields["key"] != key || fields["Content-Type"] != "image/png" {
		t.Errorf("fields key/Content-Type = %q/%q", fields["key"], fields["Content-Type"])
	}
	for _, f := range []string{"policy", "X-Amz-Signature", "X-Amz-Credential", "X-Amz-Algorithm", "X-Amz-Date"} {
		if fields[f] == "" {
			t.Errorf("missing form field %q", f)
		}
	}

	// The signed policy is what S3 enforces, so check it says what we think it says.
	policy, _ := json.Marshal(decodePolicy(t, fields))
	for _, want := range []string{
		`["content-length-range",1,5242880]`,
		`{"Content-Type":"image/png"}`,
		`{"key":"` + key + `"}`,
		`{"bucket":"` + testBucket + `"}`,
	} {
		if !strings.Contains(string(policy), want) {
			t.Errorf("policy %s\nis missing %s", policy, want)
		}
	}
}

func TestPresignDownloadURL(t *testing.T) {
	key := "users/user-1/media/6730a1b2c3d4e5f601234567.png"
	raw, err := realPresigner().PresignDownload(context.Background(), key, 15*time.Minute)
	if err != nil {
		t.Fatalf("PresignDownload() error = %v", err)
	}
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("URL %q: %v", raw, err)
	}
	if u.Scheme != "https" || !strings.HasSuffix(u.Path, key) {
		t.Errorf("URL = %q, want https and a path ending in the key", raw)
	}
	if got := u.Query().Get("X-Amz-Expires"); got != "900" {
		t.Errorf("X-Amz-Expires = %q, want 900 (15 minutes)", got)
	}
	if u.Query().Get("X-Amz-Signature") == "" {
		t.Error("URL is not signed")
	}
}

// failingPresign returns cause from every call.
type failingPresign struct{ cause error }

func (f failingPresign) PresignPostObject(context.Context, *awss3.PutObjectInput, ...func(*awss3.PresignPostOptions)) (*awss3.PresignedPostRequest, error) {
	return nil, f.cause
}

func (f failingPresign) PresignGetObject(context.Context, *awss3.GetObjectInput, ...func(*awss3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	return nil, f.cause
}

func TestPresignErrorsKeepTheirCause(t *testing.T) {
	cause := errors.New("no credentials")
	p := NewPresigner(failingPresign{cause: cause}, testBucket)

	if _, _, err := p.PresignUpload(context.Background(), "users/u/media/x.png", "image/png", 1, time.Minute); !errors.Is(err, cause) {
		t.Errorf("PresignUpload() error = %v, want it to wrap %v", err, cause)
	}
	if _, err := p.PresignDownload(context.Background(), "users/u/media/x.png", time.Minute); !errors.Is(err, cause) {
		t.Errorf("PresignDownload() error = %v, want it to wrap %v", err, cause)
	}
}
