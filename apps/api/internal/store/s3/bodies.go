// Package s3 stores entry bodies in a private, KMS-encrypted S3 bucket.
package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

// API is the part of the S3 client this package uses, so tests can pass a fake.
type API interface {
	PutObject(ctx context.Context, in *awss3.PutObjectInput, optFns ...func(*awss3.Options)) (*awss3.PutObjectOutput, error)
	GetObject(ctx context.Context, in *awss3.GetObjectInput, optFns ...func(*awss3.Options)) (*awss3.GetObjectOutput, error)
}

// Bodies implements journal.BodyStore on S3.
type Bodies struct {
	client    API
	bucket    string
	kmsKeyARN string
	maxBytes  int64
}

// NewBodies stores bodies in bucket, encrypted with the KMS key kmsKeyARN.
func NewBodies(client API, bucket, kmsKeyARN string) *Bodies {
	return &Bodies{client: client, bucket: bucket, kmsKeyARN: kmsKeyARN, maxBytes: journal.MaxBodyBytes}
}

// PutBody writes body at key. It asks for SSE-KMS with our key explicitly, so the object is
// encrypted correctly even if someone changes the bucket's default encryption.
func (b *Bodies) PutBody(ctx context.Context, key string, body []byte) error {
	_, err := b.client.PutObject(ctx, &awss3.PutObjectInput{
		Bucket:               aws.String(b.bucket),
		Key:                  aws.String(key),
		Body:                 bytes.NewReader(body),
		ContentLength:        aws.Int64(int64(len(body))),
		ContentType:          aws.String("application/json"),
		ServerSideEncryption: types.ServerSideEncryptionAwsKms,
		SSEKMSKeyId:          aws.String(b.kmsKeyARN),
		BucketKeyEnabled:     aws.Bool(true),
	})
	if err != nil {
		return fmt.Errorf("put %s: %w", key, err)
	}
	return nil
}

// GetBody reads the body at key, refusing objects larger than journal.MaxBodyBytes.
func (b *Bodies) GetBody(ctx context.Context, key string) ([]byte, error) {
	out, err := b.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", key, err)
	}
	defer out.Body.Close()

	// Read one byte past the limit: if we get it, the object is too big.
	data, err := io.ReadAll(io.LimitReader(out.Body, b.maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", key, err)
	}
	if int64(len(data)) > b.maxBytes {
		return nil, fmt.Errorf("get %s: object is larger than %d bytes", key, b.maxBytes)
	}
	return data, nil
}
