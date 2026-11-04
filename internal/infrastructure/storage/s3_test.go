package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestS3UploadDeleteAndPrivateDownloadURL(t *testing.T) {
	var put, deleted bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/employer-files/private/id.pdf" {
			t.Errorf("unexpected path %s", r.URL.Path)
			w.WriteHeader(400)
			return
		}
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			if string(body) != "%PDF-test" || r.Header.Get("Content-Type") != "application/pdf" {
				t.Error("upload body or content type changed")
			}
			if r.Header.Get("x-amz-acl") == "public-read" {
				t.Error("document uploaded publicly")
			}
			put = true
			w.Header().Set("ETag", "\"test\"")
		case http.MethodDelete:
			deleted = true
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(400)
		}
	}))
	defer server.Close()
	client := s3.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""), HTTPClient: server.Client(), RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired}, func(options *s3.Options) { options.BaseEndpoint = aws.String(server.URL); options.UsePathStyle = true })
	storage := &S3Storage{client: client, presigner: s3.NewPresignClient(client), bucket: "employer-files"}
	object, err := storage.Put(context.Background(), "private/id.pdf", strings.NewReader("%PDF-test"), "application/pdf", 9)
	if err != nil {
		t.Fatal(err)
	}
	if !put || object.Key != "private/id.pdf" || object.Size != 9 {
		t.Fatal("object metadata not preserved")
	}
	link, err := storage.PresignGet(context.Background(), object.Key, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Query().Get("X-Amz-Signature") == "" || parsed.Query().Get("X-Amz-Expires") != "300" {
		t.Fatal("download URL is unsigned or has wrong expiry")
	}
	if err := storage.Delete(context.Background(), object.Key); err != nil {
		t.Fatal(err)
	}
	if !deleted {
		t.Fatal("delete not sent")
	}
}

func TestS3MissingConfiguration(t *testing.T) {
	storage := &S3Storage{}
	if _, err := storage.Put(context.Background(), "key", strings.NewReader("x"), "text/plain", 1); err == nil {
		t.Fatal("unconfigured storage accepted upload")
	}
	if _, err := storage.PresignGet(context.Background(), "key", time.Minute); err == nil {
		t.Fatal("unconfigured storage returned URL")
	}
}
