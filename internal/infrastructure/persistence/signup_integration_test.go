package persistence_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

type capturedSMS struct {
	code string
	err  error
}

func (s *capturedSMS) SendOTP(_ context.Context, _, code string) error { s.code = code; return s.err }

// This opt-in test runs against configured PostgreSQL and rolls back every fixture.
func TestPostgresSignupVerificationAndLogin(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1 to test the configured PostgreSQL database")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	db := persistence.NewPostgresDatabase()
	rollback := errors.New("rollback test fixtures")
	err := db.Transaction(func(tx *gorm.DB) error {
		uow := persistence.NewUnitOfWork(tx)
		hasher, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("integration-test-key-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		sms := &capturedSMS{}
		signup := service.NewJobSeekerService(uow, sms, hasher)
		auth := service.NewJobSeekerAuthService(uow, hasher, tokens)
		ctx := context.Background()
		phone, nationalCode, err := fixtureIdentity()
		if err != nil {
			return err
		}
		in := usecase.SignupJobSeeker{FirstName: "Integration", LastName: "Test", NationalCode: nationalCode, City: entities.CityTehran, Address: "Test address", BirthDate: entities.JalaliDate{Year: 1380, Month: 1, Day: 1}, Gender: entities.GenderMale, Interests: []entities.Interest{entities.InterestProgramming}, Phone: phone, Password: "integration-password"}
		if err := signup.Signup(ctx, in); err != nil {
			return err
		}
		repo := uow.Factory().JobSeekerRepository()
		pending, err := repo.GetOTP(ctx, in.Phone)
		if err != nil {
			return err
		}
		if pending.Registration == nil || !hasher.Matches(pending.Registration.PasswordHash, in.Password) {
			t.Fatal("pending JSONB profile/hash did not survive database round trip")
		}
		if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: in.Phone, Password: in.Password}); !errors.Is(err, ports.ErrInvalidCredentials) {
			t.Fatalf("pending account logged in: %v", err)
		}
		if _, err := signup.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: "wrong"}); !errors.Is(err, ports.ErrInvalidOTP) {
			t.Fatal("wrong OTP accepted")
		}
		pending, err = repo.GetOTP(ctx, in.Phone)
		if err != nil {
			return err
		}
		if pending.Attempts != 1 {
			t.Fatal("failed OTP attempt rolled back")
		}
		verify := usecase.VerifySignupOTP{Phone: in.Phone, OTP: sms.code}
		seeker, err := signup.VerifyOTP(ctx, verify)
		if err != nil {
			return err
		}
		if seeker.PhoneVerifiedAt.IsZero() || seeker.BirthDate != in.BirthDate {
			t.Fatal("invalid persisted profile")
		}
		if _, err := signup.VerifyOTP(ctx, verify); !errors.Is(err, ports.ErrInvalidOTP) {
			t.Fatal("OTP replay accepted")
		}
		login, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: in.Phone, Password: in.Password})
		if err != nil {
			return err
		}
		current, err := auth.Authenticate(ctx, login.AccessToken.Value)
		if err != nil {
			return err
		}
		if current.ID != seeker.ID {
			t.Fatal("JWT authenticated wrong account")
		}
		gin.SetMode(gin.TestMode)
		router := gin.New()
		controllers := &wire.Controllers{GeneralControllers: &wire.GeneralControllers{
			JobSeekerController:     general.NewJobSeekerController(signup),
			JobSeekerAuthController: general.NewJobSeekerAuthController(auth),
		}}
		routes.SetUpRoutes(router, &wire.Application{Controllers: controllers, JobSeekerAuthService: auth})
		loginBody, err := json.Marshal(usecase.LoginJobSeeker{Phone: in.Phone, Password: in.Password})
		if err != nil {
			return err
		}
		response := callAPI(router, http.MethodPost, "/v1/job-seekers/login", string(loginBody), "")
		if response.Code != 200 {
			t.Fatalf("HTTP login returned %d", response.Code)
		}
		var credentials struct {
			AccessToken string `json:"access_token"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &credentials); err != nil {
			return err
		}
		if credentials.AccessToken == "" {
			t.Fatal("HTTP login omitted token")
		}
		for _, token := range []string{login.AccessToken.Value, credentials.AccessToken} {
			if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", "Bearer "+token); r.Code != 200 {
				t.Fatalf("fresh login token rejected: %d", r.Code)
			}
		}
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", credentials.AccessToken); r.Code != 401 {
			t.Fatal("bare authorization token must be rejected by API")
		}
		for i := 0; i < 2; i++ {
			if r := callAPI(router, http.MethodPost, "/v1/job-seekers/logout", "", "Bearer "+credentials.AccessToken); r.Code != 204 {
				t.Fatalf("logout returned %d", r.Code)
			}
		}
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", "Bearer "+credentials.AccessToken); r.Code != 401 {
			t.Fatal("logged out token accepted")
		}
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", "Bearer "+login.AccessToken.Value); r.Code != 200 {
			t.Fatal("logout revoked another login session")
		}
		otherAuth := service.NewJobSeekerAuthService(persistence.NewUnitOfWork(tx), hasher, tokens)
		if _, err := otherAuth.Authenticate(ctx, credentials.AccessToken); !errors.Is(err, ports.ErrUnauthenticated) {
			t.Fatal("revocation did not survive a new service/repository instance")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}

func callAPI(router http.Handler, method, path, body, authorization string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func fixtureIdentity() (string, string, error) {
	number, err := rand.Int(rand.Reader, big.NewInt(100000000))
	if err != nil {
		return "", "", err
	}
	phone := fmt.Sprintf("099%08d", number.Int64())
	number, err = rand.Int(rand.Reader, big.NewInt(899999999))
	if err != nil {
		return "", "", err
	}
	base := fmt.Sprintf("%09d", 100000000+number.Int64())
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(base[i]-'0') * (10 - i)
	}
	check := sum % 11
	if check >= 2 {
		check = 11 - check
	}
	return phone, fmt.Sprintf("%s%d", base, check), nil
}
