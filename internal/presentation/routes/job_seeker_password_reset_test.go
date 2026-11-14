package routes_test

import (
	"context"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

func (s *accountAPI) ForgotPassword(_ context.Context, phone string) error {
	s.calls++
	s.phone = phone
	return s.err
}
func (s *accountAPI) ResetPassword(_ context.Context, phone, otp, password string) error {
	s.calls++
	s.phone, s.code, s.password = phone, otp, password
	return s.err
}

func TestJobSeekerPublicPasswordResetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, endpoint := range []struct {
		path, body string
		status     int
	}{
		{"/v1/job-seekers/forgot-password", `{"phone":"09123456789"}`, 202},
		{"/v1/job-seekers/reset-password", `{"phone":"09123456789","otp":"123456","new_password":"new-password"}`, 200},
	} {
		for _, scenario := range []string{"success", "missing fields", "malformed", "oversized", "service error"} {
			t.Run(endpoint.path+scenario, func(t *testing.T) {
				s := &accountAPI{}
				app := &wire.Application{Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{JobSeekerController: general.NewJobSeekerController(s), JobSeekerAuthController: general.NewJobSeekerAuthController(s)}}, JobSeekerAuthService: s}
				router := gin.New()
				routes.SetUpRoutes(router, app)
				body, want := endpoint.body, endpoint.status
				switch scenario {
				case "missing fields":
					body, want = `{}`, 400
				case "malformed":
					body, want = `{`, 400
				case "oversized":
					body, want = `{"phone":"`+strings.Repeat("x", 17*1024)+`"}`, 400
				case "service error":
					s.err, want = domainerrors.ErrSMSUnavailable, 503
				}
				req := httptest.NewRequest(http.MethodPost, endpoint.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				r := httptest.NewRecorder()
				router.ServeHTTP(r, req)
				if r.Code != want {
					t.Fatalf("got %d, want %d: %s", r.Code, want, r.Body.String())
				}
				if scenario == "success" || scenario == "service error" {
					if s.calls != 1 || s.phone != "09123456789" || r.Header().Get("Cache-Control") != "no-store" {
						t.Fatal("phone not passed or response cached")
					}
					if strings.HasSuffix(endpoint.path, "reset-password") && (s.code != "123456" || s.password != "new-password") {
						t.Fatal("OTP/password not passed")
					}
					if scenario == "success" && strings.HasSuffix(endpoint.path, "reset-password") && !strings.Contains(r.Body.String(), "Password reset successfully") {
						t.Fatal("reset success message missing")
					}
				} else if s.calls != 0 {
					t.Fatal("invalid input reached service")
				}
			})
		}
	}
}
