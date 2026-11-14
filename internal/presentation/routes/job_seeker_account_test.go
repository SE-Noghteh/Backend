package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

type accountAPI struct {
	usecase.JobSeekerService
	usecase.JobSeekerAuthService
	id                    uint64
	phone, code, password string
	calls                 int
	err                   error
}

func (*accountAPI) Authenticate(_ context.Context, token string) (*entities.JobSeeker, error) {
	if token != "test-token" {
		return nil, domainerrors.ErrUnauthenticated
	}
	return &entities.JobSeeker{ID: 7}, nil
}
func (s *accountAPI) RequestPhoneChange(_ context.Context, id uint64, phone string) error {
	s.calls++
	s.id, s.phone = id, phone
	return s.err
}
func (s *accountAPI) VerifyPhoneChange(_ context.Context, id uint64, in usecase.VerifySignupOTP) (*entities.JobSeeker, error) {
	s.calls++
	s.id, s.phone, s.code = id, in.Phone, in.OTP
	return &entities.JobSeeker{ID: id, Phone: in.Phone}, s.err
}
func (s *accountAPI) DeleteAccount(_ context.Context, id uint64, password string) error {
	s.calls++
	s.id, s.password = id, password
	return s.err
}

func TestJobSeekerAccountRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, endpoint := range []struct {
		method, path, body string
		status             int
	}{
		{http.MethodPost, "/v1/job-seekers/phone-change", `{"new_phone":"09129876543"}`, 202},
		{http.MethodPost, "/v1/job-seekers/phone-change/verify", `{"new_phone":"09129876543","otp":"123456"}`, 200},
		{http.MethodDelete, "/v1/job-seekers/me", `{"password":"current-password"}`, 200},
	} {
		for _, scenario := range []string{"success", "anonymous", "invalid token", "missing field", "invalid JSON", "oversized", "service error"} {
			t.Run(endpoint.method+endpoint.path+scenario, func(t *testing.T) {
				s := &accountAPI{}
				app := &wire.Application{JobSeekerAuthService: s, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{JobSeekerController: general.NewJobSeekerController(s), JobSeekerAuthController: general.NewJobSeekerAuthController(s)}}}
				router := gin.New()
				routes.SetUpRoutes(router, app)
				body, token, want := endpoint.body, "Bearer test-token", endpoint.status
				switch scenario {
				case "anonymous":
					token, want = "", 401
				case "invalid token":
					token, want = "Bearer invalid", 401
				case "missing field":
					body, want = `{}`, 400
				case "invalid JSON":
					body, want = `{`, 400
				case "oversized":
					body, want = `{"password":"`+strings.Repeat("x", 17*1024)+`"}`, 400
				case "service error":
					s.err, want = domainerrors.ErrInvalidCredentials, 401
				}
				req := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", token)
				response := httptest.NewRecorder()
				router.ServeHTTP(response, req)
				if response.Code != want {
					t.Fatalf("got %d, want %d: %s", response.Code, want, response.Body.String())
				}
				if scenario == "success" || scenario == "service error" {
					if scenario == "success" && endpoint.method == http.MethodDelete {
						var result general.DeleteJobSeekerResponse
						if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil || result.Message != "Account deleted successfully" {
							t.Fatalf("missing deletion success message: %s", response.Body.String())
						}
					}
					if s.calls != 1 || s.id != 7 {
						t.Fatal("request did not use authenticated account")
					}
					if endpoint.method == http.MethodDelete && s.password != "current-password" {
						t.Fatal("password missing")
					}
					if endpoint.method == http.MethodPost && s.phone != "09129876543" {
						t.Fatal("new phone missing")
					}
					if strings.HasSuffix(endpoint.path, "/verify") && s.code != "123456" {
						t.Fatal("OTP missing")
					}
				} else if s.calls != 0 {
					t.Fatal("invalid/unauthenticated request reached service")
				}
			})
		}
	}
}
