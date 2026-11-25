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

func (s *resetTestSMS) SendPasswordResetOTP(_ context.Context, destination, code string) error {
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
		resetService := service.NewPasswordResetService(uow, hasher, sms)
		auth := service.NewJobSeekerAuthService(uow, hasher, tokens)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		routes.SetUpRoutes(router, &wire.Application{JobSeekerAuthService: auth, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{PasswordResetController: general.NewPasswordResetController(resetService), JobSeekerAuthController: general.NewJobSeekerAuthController(auth)}}})
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
		if r := callAPI(router, "POST", "/v1/job-seekers/forgot-password", forgotBody, ""); r.Code != 202 || sms.count != 1 {
			t.Fatal("reset resend cooldown bypassed")
		}
		resetRepo := uow.Factory().PasswordResetRepository()
		challenge, err := resetRepo.Get(ctx, enums.RoleJobSeeker, phone)
		if err != nil {
			return err
		}
		if challenge == nil || challenge.OTP.Phone != phone || challenge.OTP.CodeHash == sms.code || challenge.OTP.Salt == "" {
			t.Fatal("password reset challenge did not survive persistence or leaked plaintext")
		}
		if err := repo.SavePhoneChangeOTP(ctx, seeker.ID, &challenge.OTP); err != nil {
			return err
		}
		wrongBody := fmt.Sprintf(`{"phone":%q,"otp":"bad"}`, phone)
		if r := callAPI(router, "POST", "/v1/job-seekers/verify-reset-otp", wrongBody, ""); r.Code != 401 {
			t.Fatal("wrong reset OTP accepted")
		}
		challenge, err = resetRepo.Get(ctx, enums.RoleJobSeeker, phone)
		if err != nil {
			return err
		}
		if challenge == nil || challenge.OTP.Attempts != 1 {
			t.Fatal("wrong reset attempts rolled back")
		}
		if err := uow.Factory().LoginAttemptRepository().SaveLoginAttempt(ctx, &entities.LoginAttempt{Phone: phone, Failures: 5, WindowStartedAt: time.Now().UTC()}); err != nil {
			return err
		}
		verifyBody := fmt.Sprintf(`{"phone":%q,"otp":%q}`, phone, sms.code)
		verified := callAPI(router, "POST", "/v1/job-seekers/verify-reset-otp", verifyBody, "")
		var grant general.PasswordResetTokenResponse
		if verified.Code != 200 || json.Unmarshal(verified.Body.Bytes(), &grant) != nil || grant.ResetToken == "" {
			t.Fatalf("verify reset: %d %s", verified.Code, verified.Body.String())
		}
		if r := callAPI(router, "POST", "/v1/job-seekers/verify-reset-otp", verifyBody, ""); r.Code != 401 {
			t.Fatal("verified OTP could be reused")
		}
		resetBody := fmt.Sprintf(`{"reset_token":%q,"new_password":"new-password"}`, grant.ResetToken)
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
		if challenge, err := resetRepo.Get(ctx, enums.RoleJobSeeker, phone); err != nil || challenge != nil {
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

func TestPostgresEmployerPasswordReset(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback employer password reset fixtures")
	err := persistence.NewPostgresDatabase().Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		uow := persistence.NewUnitOfWork(tx)
		hasher, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("employer-reset-test-secret-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		phone, national, err := fixtureIdentity()
		if err != nil {
			return err
		}
		const oldPassword = "OldEmployerPass123!"
		const newPassword = "NewEmployerPass456!"
		hash, err := hasher.Hash(oldPassword)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		doc := valueobjects.StoredObject{Key: "employer-test/video.mp4", ContentType: "video/mp4", Size: 10}
		repo := uow.Factory().EmployerRepository()
		individual := &entities.IndividualEmployer{
			Phone: phone, NationalCode: national, PasswordHash: hash, PhoneVerifiedAt: now, CreatedAt: now,
			FirstName: "Individual", Gender: enums.GenderMale, BirthDate: valueobjects.JalaliDate{Year: 1375, Month: 1, Day: 1},
			IdentityDocument: doc, VerificationVideo: doc, Province: enums.ProvinceTehran, City: enums.CityTehran,
		}
		if err := repo.CreateIndividual(ctx, individual); err != nil {
			return err
		}
		legal := &entities.LegalEmployer{
			Phone: phone, CompanyNationalID: "1" + national, PasswordHash: hash, PhoneVerifiedAt: now, CreatedAt: now,
			CompanyName: "Legal", LegalEntityType: "limited_liability", RepresentativePosition: "ceo", RepresentativeGender: enums.GenderMale,
			Activities: []enums.Interest{enums.InterestDesign}, EstablishmentDocument: doc, IdentityDocument: doc,
			VerificationVideo: doc, Province: enums.ProvinceTehran, City: enums.CityTehran,
		}
		if err := repo.CreateLegal(ctx, legal); err != nil {
			return err
		}
		sms := &resetTestSMS{}
		router := gin.New()
		routes.SetUpRoutes(router, &wire.Application{Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{
			PasswordResetController:      general.NewPasswordResetController(service.NewPasswordResetService(uow, hasher, sms)),
			IndividualEmployerController: general.NewIndividualEmployerController(service.NewIndividualEmployerService(uow, nil, hasher, tokens, nil, nil)),
			LegalEmployerController:      general.NewLegalEmployerController(service.NewLegalEmployerService(uow, nil, hasher, tokens, nil, nil)),
		}}})
		login := func(group, password string) (string, int) {
			body := fmt.Sprintf(`{"phone":%q,"password":%q}`, phone, password)
			r := callAPI(router, "POST", "/v1/"+group+"/login", body, "")
			var result general.AccessTokenResponse
			if r.Code == 200 && json.Unmarshal(r.Body.Bytes(), &result) != nil {
				t.Fatalf("invalid login response for %s", group)
			}
			return result.AccessToken, r.Code
		}
		for _, pair := range [][2]string{{"individual-employers", "legal-employers"}, {"legal-employers", "individual-employers"}} {
			group, other := pair[0], pair[1]
			path := "/v1/" + group
			oldToken, status := login(group, oldPassword)
			if status != 200 || oldToken == "" {
				t.Fatalf("could not sign in as %s before reset: %d", group, status)
			}
			if r := callAPI(router, "POST", path+"/forgot-password", fmt.Sprintf(`{"phone":%q}`, phone), ""); r.Code != 202 {
				t.Fatalf("forgot password for %s: %d %s", group, r.Code, r.Body)
			}
			r := callAPI(router, "POST", path+"/verify-reset-otp", fmt.Sprintf(`{"phone":%q,"otp":%q}`, phone, sms.code), "")
			var grant general.PasswordResetTokenResponse
			if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &grant) != nil || grant.ResetToken == "" {
				t.Fatalf("verify reset for %s: %d %s", group, r.Code, r.Body)
			}
			weak := fmt.Sprintf(`{"reset_token":%q,"new_password":"alllowercase"}`, grant.ResetToken)
			if r := callAPI(router, "POST", path+"/reset-password", weak, ""); r.Code != 400 {
				t.Fatalf("weak password accepted for %s: %d", group, r.Code)
			}
			reset := fmt.Sprintf(`{"reset_token":%q,"new_password":%q}`, grant.ResetToken, newPassword)
			if r := callAPI(router, "POST", "/v1/"+other+"/reset-password", reset, ""); r.Code != 401 {
				t.Fatalf("reset token accepted for another employer role: %d", r.Code)
			}
			if r := callAPI(router, "POST", path+"/reset-password", reset, ""); r.Code != 200 {
				t.Fatalf("reset for %s: %d %s", group, r.Code, r.Body)
			}
			if r := callAPI(router, "GET", path+"/me", "", "Bearer "+oldToken); r.Code != 401 {
				t.Fatalf("old session survived reset for %s: %d", group, r.Code)
			}
			if r := callAPI(router, "POST", path+"/reset-password", reset, ""); r.Code != 401 {
				t.Fatalf("reset token reusable for %s: %d", group, r.Code)
			}
			if _, status := login(group, oldPassword); status != 401 {
				t.Fatalf("old password worked after reset for %s: %d", group, status)
			}
			freshToken, status := login(group, newPassword)
			if status != 200 || freshToken == "" {
				t.Fatalf("new password failed for %s: %d", group, status)
			}
			if r := callAPI(router, "GET", path+"/me", "", "Bearer "+freshToken); r.Code != 200 {
				t.Fatalf("new session rejected for %s: %d %s", group, r.Code, r.Body)
			}
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
