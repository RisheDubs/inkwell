package s3

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/RisheDubs/inkwell/apps/api/internal/journal"
)

const (
	testBucket = "inkwell-dev-content-123456789012"
	testKeyARN = "arn:aws:kms:ap-southeast-2:123456789012:key/11111111-2222-3333-4444-555555555555"
	testKey    = "users/user-1/entries/e1.json"
)

// fakeS3 records the last request and returns a canned object or error.
type fakeS3 struct {
	put     *awss3.PutObjectInput
	putBody string
	get     *awss3.GetObjectInput
	object  string
	err     error
}

func (f *fakeS3) PutObject(_ context.Context, in *awss3.PutObjectInput, _ ...func(*awss3.Options)) (*awss3.PutObjectOutput, error) {
	f.put = in
	b, _ := io.ReadAll(in.Body)
	f.putBody = string(b)
	if f.err != nil {
		return nil, f.err
	}
	return &awss3.PutObjectOutput{}, nil
}

func (f *fakeS3) GetObject(_ context.Context, in *awss3.GetObjectInput, _ ...func(*awss3.Options)) (*awss3.GetObjectOutput, error) {
	f.get = in
	if f.err != nil {
		return nil, f.err
	}
	return &awss3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(f.object))}, nil
}

func TestPutBodyEncryptsWithOurKey(t *testing.T) {
	fake := &fakeS3{}
	body := `{"type":"doc","content":[]}`
	if err := NewBodies(fake, testBucket, testKeyARN).PutBody(context.Background(), testKey, []byte(body)); err != nil {
		t.Fatalf("PutBody() error = %v", err)
	}

	in := fake.put
	if aws.ToString(in.Bucket) != testBucket || aws.ToString(in.Key) != testKey {
		t.Errorf("wrote to %s/%s, want %s/%s", aws.ToString(in.Bucket), aws.ToString(in.Key), testBucket, testKey)
	}
	if in.ServerSideEncryption != types.ServerSideEncryptionAwsKms {
		t.Errorf("ServerSideEncryption = %q, want aws:kms", in.ServerSideEncryption)
	}
	if aws.ToString(in.SSEKMSKeyId) != testKeyARN {
		t.Errorf("SSEKMSKeyId = %q, want %q", aws.ToString(in.SSEKMSKeyId), testKeyARN)
	}
	if !aws.ToBool(in.BucketKeyEnabled) {
		t.Error("BucketKeyEnabled = false, want true")
	}
	if aws.ToString(in.ContentType) != "application/json" {
		t.Errorf("ContentType = %q, want application/json", aws.ToString(in.ContentType))
	}
	if fake.putBody != body {
		t.Errorf("uploaded %q, want %q", fake.putBody, body)
	}
}

func TestGetBody(t *testing.T) {
	fake := &fakeS3{object: `{"type":"doc"}`}
	got, err := NewBodies(fake, testBucket, testKeyARN).GetBody(context.Background(), testKey)
	if err != nil {
		t.Fatalf("GetBody() error = %v", err)
	}
	if string(got) != `{"type":"doc"}` {
		t.Errorf("GetBody() = %q", got)
	}
	if aws.ToString(fake.get.Key) != testKey {
		t.Errorf("read key %q, want %q", aws.ToString(fake.get.Key), testKey)
	}
}

func TestGetBodyRejectsOversizedObjects(t *testing.T) {
	fake := &fakeS3{object: strings.Repeat("x", journal.MaxBodyBytes+1)}
	if _, err := NewBodies(fake, testBucket, testKeyARN).GetBody(context.Background(), testKey); err == nil {
		t.Error("GetBody() of an oversized object: want an error, got nil")
	}
}

func TestErrorsKeepTheirCause(t *testing.T) {
	cause := errors.New("AccessDenied")
	b := NewBodies(&fakeS3{err: cause}, testBucket, testKeyARN)

	if err := b.PutBody(context.Background(), testKey, []byte("{}")); !errors.Is(err, cause) {
		t.Errorf("PutBody() error = %v, want it to wrap %v", err, cause)
	}
	if _, err := b.GetBody(context.Background(), testKey); !errors.Is(err, cause) {
		t.Errorf("GetBody() error = %v, want it to wrap %v", err, cause)
	}
}
