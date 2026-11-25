package general

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
)

type capturedJobSeekerSignup struct {
	usecase.JobSeekerService
	input    usecase.SignupJobSeeker
	video    []byte
	identity []byte
	calls    int
}

func (s *capturedJobSeekerSignup) Signup(_ context.Context, in usecase.SignupJobSeeker) error {
	s.input = in
	s.calls++
	var err error
	s.video, err = io.ReadAll(in.VerificationVideo.Body)
	if err != nil {
		return err
	}
	s.identity, err = io.ReadAll(in.IdentityDocument.Body)
	return err
}
func (*capturedJobSeekerSignup) RequestOTP(context.Context, string) error { return nil }
func (*capturedJobSeekerSignup) VerifyOTP(context.Context, usecase.VerifySignupOTP) (*entities.JobSeeker, error) {
	return nil, nil
}
func (*capturedJobSeekerSignup) VideoURL(context.Context, uint64) (string, error) { return "", nil }

func TestJobSeekerSignupMultipartDocuments(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, scenario := range []struct {
		name           string
		videos, images int
	}{
		{"missing video", 0, 1},
		{"duplicate video", 2, 1},
		{"missing image", 1, 0},
		{"duplicate image", 1, 2},
		{"one of each", 1, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for name, value := range map[string]string{"first_name": "Test", "birth_year": "1380", "birth_month": "2", "birth_day": "3", "province": string(enums.ProvinceTehran), "postal_code": "0123456789"} {
				if err := writer.WriteField(name, value); err != nil {
					t.Fatal(err)
				}
			}
			for _, interest := range []string{"programming", "design"} {
				if err := writer.WriteField("interests", interest); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < scenario.videos; i++ {
				file, err := writer.CreateFormFile("verification_video", "video.mp4")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("video bytes")); err != nil {
					t.Fatal(err)
				}
			}
			for i := 0; i < scenario.images; i++ {
				file, err := writer.CreateFormFile("identity_document", "identity.png")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("image bytes")); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			s := &capturedJobSeekerSignup{}
			router := gin.New()
			router.POST("/signup", NewJobSeekerController(s).Signup)
			req := httptest.NewRequest(http.MethodPost, "/signup", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if scenario.videos != 1 || scenario.images != 1 {
				if response.Code != 400 || s.calls != 0 {
					t.Fatal("missing or duplicate document accepted")
				}
				return
			}
			if response.Code != 202 || s.calls != 1 || string(s.video) != "video bytes" || s.input.VerificationVideo.Name != "video.mp4" || string(s.identity) != "image bytes" {
				t.Fatal("multipart documents were not passed to signup")
			}
			if s.input.Province != enums.ProvinceTehran || s.input.PostalCode != "0123456789" {
				t.Fatal("province or postal code was not bound correctly")
			}
			if s.input.BirthDate.Year != 1380 || s.input.BirthDate.Month != 2 || s.input.BirthDate.Day != 3 || len(s.input.Interests) != 2 {
				t.Fatal("profile fields were not bound correctly")
			}
		})
	}
}

func TestJobSeekerSignupRejectsJSONAndOversizedBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, scenario := range []string{"json", "oversized"} {
		t.Run(scenario, func(t *testing.T) {
			s := &capturedJobSeekerSignup{}
			router := gin.New()
			router.POST("/signup", NewJobSeekerController(s).Signup)
			var req *http.Request
			want := http.StatusBadRequest
			if scenario == "json" {
				req = httptest.NewRequest(http.MethodPost, "/signup", bytes.NewBufferString(`{"phone":"09123456789"}`))
				req.Header.Set("Content-Type", "application/json")
			} else {
				var body bytes.Buffer
				writer := multipart.NewWriter(&body)
				file, err := writer.CreateFormFile("verification_video", "video.mp4")
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write(make([]byte, 57*1024*1024)); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				req = httptest.NewRequest(http.MethodPost, "/signup", &body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
				want = http.StatusRequestEntityTooLarge
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != want || s.calls != 0 {
				t.Fatalf("invalid request reached signup: status %d", response.Code)
			}
		})
	}
}
