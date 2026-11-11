package s3

import (
	"context"
	"fmt"
	"maps"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// PresignAPI is the part of the S3 presign client this package uses, so tests can pass a fake.
type PresignAPI interface {
	PresignPostObject(ctx context.Context, in *awss3.PutObjectInput, optFns ...func(*awss3.PresignPostOptions)) (*awss3.PresignedPostRequest, error)
	PresignGetObject(ctx context.Context, in *awss3.GetObjectInput, optFns ...func(*awss3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

// Presigner implements journal.Presigner on S3. Signing happens locally with the API's
// credentials: no request reaches AWS until the client uses the link.
type Presigner struct {
	client PresignAPI
	bucket string
}

// NewPresigner signs links for objects in bucket.
func NewPresigner(client PresignAPI, bucket string) *Presigner {
	return &Presigner{client: client, bucket: bucket}
}

// PresignUpload returns a presigned POST. Its signed policy only accepts an object at
// exactly key, with exactly contentType, between 1 byte and maxBytes, until expires.
// Anything else and S3 itself rejects the upload.
func (p *Presigner) PresignUpload(ctx context.Context, key, contentType string, maxBytes int64, expires time.Duration) (string, map[string]string, error) {
	out, err := p.client.PresignPostObject(ctx, &awss3.PutObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}, func(o *awss3.PresignPostOptions) {
		o.Expires = expires
		o.Conditions = []any{
			[]any{"content-length-range", 1, maxBytes},
			map[string]string{"Content-Type": contentType},
		}
	})
	if err != nil {
		return "", nil, fmt.Errorf("presign upload %s: %w", key, err)
	}

	// The client must send every policy field back unchanged, plus the Content-Type it promised.
	fields := make(map[string]string, len(out.Values)+1)
	maps.Copy(fields, out.Values)
	fields["Content-Type"] = contentType
	return out.URL, fields, nil
}

// PresignDownload returns a GET URL for key that works until expires.
func (p *Presigner) PresignDownload(ctx context.Context, key string, expires time.Duration) (string, error) {
	out, err := p.client.PresignGetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(p.bucket),
		Key:    aws.String(key),
	}, awss3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("presign download %s: %w", key, err)
	}
	return out.URL, nil
}
