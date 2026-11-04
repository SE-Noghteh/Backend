package storage

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/joho/godotenv"
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

func TestS3PublicDownloadEndpoint(t *testing.T) {
	client := s3.NewFromConfig(aws.Config{
		Region: "us-east-1", Credentials: credentials.NewStaticCredentialsProvider("test", "test", ""),
	}, func(options *s3.Options) {
		options.BaseEndpoint = aws.String("http://s3:9000")
		options.UsePathStyle = true
	})
	storage := &S3Storage{client: client, presigner: newPresigner(client, "http://localhost:9000"), bucket: "employer-files"}
	link, err := storage.PresignGet(context.Background(), "private/id.pdf", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(link)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Host != "localhost:9000" || parsed.Path != "/employer-files/private/id.pdf" || parsed.Query().Get("X-Amz-Signature") == "" {
		t.Fatal("download URL must be signed for the browser endpoint")
	}
	if aws.ToString(client.Options().BaseEndpoint) != "http://s3:9000" {
		t.Fatal("presigning changed the upload endpoint")
	}
}

func TestLocalS3Integration(t *testing.T) {
	if os.Getenv("RUN_LOCAL_S3_INTEGRATION") != "1" {
		t.Skip("set RUN_LOCAL_S3_INTEGRATION=1 after starting local storage")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	storage, err := NewS3Storage()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	key := "storage-smoke/" + time.Now().UTC().Format("20060102T150405.000000000") + ".txt"
	const content = "local S3 storage smoke test"
	if _, err := storage.Put(ctx, key, strings.NewReader(content), "text/plain", int64(len(content))); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := storage.Delete(cleanupCtx, key); err != nil {
			t.Error(err)
		}
	})
	link, err := storage.PresignGet(ctx, key, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	httpClient := &http.Client{Timeout: 10 * time.Second}
	response, err := httpClient.Get(link)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil || response.StatusCode != http.StatusOK || string(body) != content {
		t.Fatalf("signed download failed: status=%d error=%v", response.StatusCode, err)
	}
	anonymousURL, _ := url.Parse(link)
	anonymousURL.RawQuery = ""
	response, err = httpClient.Get(anonymousURL.String())
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatalf("anonymous download must be forbidden, got %d", response.StatusCode)
	}
	if err := storage.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	response, err = httpClient.Get(link)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("deleted object still exists, got %d", response.StatusCode)
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
