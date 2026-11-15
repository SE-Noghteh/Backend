package persistence_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"os"
	"strings"
	"testing"
	"time"
	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

type resetTestSMS struct {
	destination, code string
	count             int
}

func (s *resetTestSMS) SendOTP(_ context.Context, destination, code string) error {
	s.destination, s.code = destination, code
	s.count++
	return nil
}

func TestPostgresJobSeekerPasswordReset(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	db := persistence.NewPostgresDatabase()
	rollback := errors.New("rollback password reset fixtures")
	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		uow := persistence.NewUnitOfWork(tx)
		hasher, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("password-reset-test-secret-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		sms := &resetTestSMS{}
		storage := &employerTestStorage{objects: map[string][]byte{}}
		signup := service.NewJobSeekerService(uow, sms, hasher, storage, parentIntegrationMedia{})
		auth := service.NewJobSeekerAuthService(uow, hasher, tokens)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		routes.SetUpRoutes(router, &wire.Application{JobSeekerAuthService: auth, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{JobSeekerController: general.NewJobSeekerController(signup), JobSeekerAuthController: general.NewJobSeekerAuthController(auth)}}})
		phone, national, err := fixtureIdentity()
		if err != nil {
			return err
		}
		hash, err := hasher.Hash("old-password")
		if err != nil {
			return err
		}
		seeker := &entities.JobSeeker{FirstName: "Reset", LastName: "Test", NationalCode: national, Phone: phone, PasswordHash: hash, City: enums.CityTehran, Address: "Test address", BirthDate: valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 20, Month: 1, Day: 1}, Gender: enums.GenderMale, Interests: []enums.Interest{enums.InterestProgramming}, PhoneVerifiedAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), AdminReview: valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewApproved}}
		repo := uow.Factory().JobSeekerRepository()
		if err := repo.Create(ctx, seeker); err != nil {
			return err
		}
		var oldTokens []string
		for i := 0; i < 2; i++ {
			login, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: phone, Password: "old-password"})
			if err != nil {
				return err
			}
			oldTokens = append(oldTokens, login.AccessToken.Value)
		}
		forgotBody := fmt.Sprintf(`{"phone":%q}`, phone)
		response := callAPI(router, "POST", "/v1/job-seekers/forgot-password", forgotBody, "")
		if response.Code != 202 || sms.destination != "98"+phone[1:] || len(sms.code) != 6 {
			t.Fatalf("reset OTP not sent: %d %s", response.Code, response.Body.String())
		}
		unknown, _, err := fixtureIdentity()
		if err != nil {
			return err
		}
		if r := callAPI(router, "POST", "/v1/job-seekers/forgot-password", fmt.Sprintf(`{"phone":%q}`, unknown), ""); r.Code != 202 || r.Body.String() != response.Body.String() || sms.count != 1 {
			t.Fatal("unknown account response differs or sent SMS")
		}
		if r := callAPI(router, "POST", "/v1/job-seekers/forgot-password", forgotBody, ""); r.Code != 429 {
			t.Fatal("reset resend cooldown bypassed")
		}
		otp, err := repo.GetPasswordResetOTP(ctx, seeker.ID)
		if err != nil {
			return err
		}
		if otp == nil || otp.Phone != phone || otp.CodeHash == sms.code || otp.Salt == "" {
			t.Fatal("password reset challenge did not survive persistence or leaked plaintext")
		}
		if err := repo.SavePhoneChangeOTP(ctx, seeker.ID, otp); err != nil {
			return err
		}
		wrongBody := fmt.Sprintf(`{"phone":%q,"otp":"bad","new_password":"new-password"}`, phone)
		if r := callAPI(router, "POST", "/v1/job-seekers/reset-password", wrongBody, ""); r.Code != 401 {
			t.Fatal("wrong reset OTP accepted")
		}
		otp, err = repo.GetPasswordResetOTP(ctx, seeker.ID)
		if err != nil {
			return err
		}
		if otp.Attempts != 1 {
			t.Fatal("wrong reset attempts rolled back")
		}
		if err := uow.Factory().LoginAttemptRepository().SaveLoginAttempt(ctx, &entities.LoginAttempt{Phone: phone, Failures: 5, WindowStartedAt: time.Now().UTC()}); err != nil {
			return err
		}
		resetBody := fmt.Sprintf(`{"phone":%q,"otp":%q,"new_password":"new-password"}`, phone, sms.code)
		if r := callAPI(router, "POST", "/v1/job-seekers/reset-password", resetBody, ""); r.Code != 200 || !strings.Contains(r.Body.String(), "Password reset successfully") {
			t.Fatalf("reset failed: %d %s", r.Code, r.Body.String())
		}
		persisted, err := repo.FindByID(ctx, seeker.ID)
		if err != nil {
			return err
		}
		if !hasher.Matches(persisted.PasswordHash, "new-password") || hasher.Matches(persisted.PasswordHash, "old-password") || persisted.SessionVersion != 1 || persisted.AdminApprovalStatus != enums.AdminReviewApproved {
			t.Fatal("credentials not changed or approval was lost")
		}
		if otp, err := repo.GetPasswordResetOTP(ctx, seeker.ID); err != nil || otp != nil {
			t.Fatal("reset OTP was not consumed", err)
		}
		if otp, err := repo.GetPhoneChangeOTP(ctx, seeker.ID); err != nil || otp != nil {
			t.Fatal("pending phone-change OTP survived reset", err)
		}
		if attempts, err := uow.Factory().LoginAttemptRepository().GetLoginAttempt(ctx, phone); err != nil || attempts != nil {
			t.Fatal("reset did not clear login throttle", err)
		}
		for _, token := range oldTokens {
			if r := callAPI(router, "GET", "/v1/job-seekers/me", "", "Bearer "+token); r.Code != 401 {
				t.Fatal("old token survived reset")
			}
		}
		if r := callAPI(router, "POST", "/v1/job-seekers/reset-password", resetBody, ""); r.Code != 401 {
			t.Fatal("reset OTP replay accepted")
		}
		if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: phone, Password: "old-password"}); err == nil {
			t.Fatal("old password accepted")
		}
		loginResponse := callAPI(router, "POST", "/v1/job-seekers/login", fmt.Sprintf(`{"phone":%q,"password":"new-password"}`, phone), "")
		if loginResponse.Code != 200 {
			t.Fatalf("new password login failed: %s", loginResponse.Body.String())
		}
		var login general.LoginResponse
		if err := json.Unmarshal(loginResponse.Body.Bytes(), &login); err != nil {
			return err
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", "Bearer "+login.AccessToken); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"approved"`) {
			t.Fatal("new session rejected or approval changed")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
