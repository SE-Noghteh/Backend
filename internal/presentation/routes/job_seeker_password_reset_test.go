package routes_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

type resetAPI struct {
	role                                 enums.AccountRole
	method, phone, code, token, password string
	err                                  error
}

func (s *resetAPI) Request(_ context.Context, role enums.AccountRole, phone string) error {
	s.role, s.method, s.phone = role, "request", phone
	return s.err
}

func (s *resetAPI) VerifyOTP(_ context.Context, role enums.AccountRole, phone, code string) (string, error) {
	s.role, s.method, s.phone, s.code = role, "verify", phone, code
	return "one-time-token", s.err
}

func (s *resetAPI) Reset(_ context.Context, role enums.AccountRole, token, password string) error {
	s.role, s.method, s.token, s.password = role, "reset", token, password
	return s.err
}

func TestPublicPasswordResetRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	roles := map[string]enums.AccountRole{
		"job-seekers": enums.RoleJobSeeker, "parents": enums.RoleParent,
		"individual-employers": enums.RoleIndividualEmployer, "legal-employers": enums.RoleLegalEmployer,
	}
	for group, role := range roles {
		for _, endpoint := range []struct {
			path, method, body string
			status             int
		}{
			{"forgot-password", "request", `{"phone":"09123456789"}`, 202},
			{"verify-reset-otp", "verify", `{"phone":"09123456789","otp":"123456"}`, 200},
			{"reset-password", "reset", `{"reset_token":"one-time-token","new_password":"new-password"}`, 200},
		} {
			t.Run(group+"/"+endpoint.path, func(t *testing.T) {
				service := &resetAPI{}
				app := &wire.Application{Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{PasswordResetController: general.NewPasswordResetController(service)}}}
				router := gin.New()
				routes.SetUpRoutes(router, app)
				path := "/v1/" + group + "/" + endpoint.path
				send := func(body string) *httptest.ResponseRecorder {
					req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
					req.Header.Set("Content-Type", "application/json")
					response := httptest.NewRecorder()
					router.ServeHTTP(response, req)
					return response
				}
				for _, body := range []string{`{}`, `{`, `{"phone":"` + strings.Repeat("x", 17*1024) + `"}`} {
					if response := send(body); response.Code != 400 || service.method != "" {
						t.Fatalf("invalid input reached service: %d %s", response.Code, response.Body.String())
					}
				}
				response := send(endpoint.body)
				if response.Code != endpoint.status || service.role != role || service.method != endpoint.method || response.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("incorrect route or response: %d %s", response.Code, response.Body.String())
				}
				if endpoint.method == "verify" && (service.phone != "09123456789" || service.code != "123456" || !strings.Contains(response.Body.String(), `"reset_token":"one-time-token"`)) {
					t.Fatal("verification did not accept OTP or return token")
				}
				if endpoint.method == "reset" && (service.token != "one-time-token" || service.password != "new-password" || send(`{"phone":"09123456789","otp":"123456","new_password":"new-password"}`).Code != 400) {
					t.Fatal("reset did not require the new contract")
				}
				service.err = domainerrors.ErrInvalidResetToken
				if response := send(endpoint.body); response.Code != 401 {
					t.Fatalf("service error translated to %d", response.Code)
				}
			})
		}
	}
}
