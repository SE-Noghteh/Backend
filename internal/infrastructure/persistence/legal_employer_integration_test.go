package persistence_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/media"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/persistence/models"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

// Real PostgreSQL, bcrypt, JWT, multipart binding, and ffprobe; SMS/storage are local fakes.
func TestPostgresLegalEmployerSingleSubmission(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	videoPath := filepath.Join(t.TempDir(), "identity.mp4")
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=16x16:r=1", "-t", "2", "-c:v", "mpeg4", "-y", videoPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("video fixture: %v %s", err, output)
	}
	video, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	db := persistence.NewPostgresDatabase()
	rollback := errors.New("rollback employer fixtures")
	err = db.Transaction(func(tx *gorm.DB) error {
		uow := persistence.NewUnitOfWork(tx)
		passwords, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("integration-employer-key-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		sms := &capturedSMS{}
		storage := &employerTestStorage{objects: map[string][]byte{}}
		employers := service.NewLegalEmployerService(uow, sms, passwords, tokens, storage, media.NewValidator())
		jobAuth := service.NewJobSeekerAuthService(uow, passwords, tokens)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		routes.SetUpRoutes(router, &wire.Application{JobSeekerAuthService: jobAuth, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{
			LegalEmployerController:      general.NewLegalEmployerController(employers),
			IndividualEmployerController: general.NewIndividualEmployerController(service.NewIndividualEmployerService(uow, sms, passwords, tokens, storage, media.NewValidator())),
			JobSeekerController:          general.NewJobSeekerController(nil), JobSeekerAuthController: general.NewJobSeekerAuthController(jobAuth),
		}}})
		phone, national, err := fixtureIdentity()
		if err != nil {
			return err
		}
		body, _ := json.Marshal(general.ResendOTPRequest{Phone: phone})
		if response := callAPI(router, http.MethodPost, "/v1/legal-employers/otp", string(body), ""); response.Code != 400 {
			t.Fatalf("OTP before signup returned %d", response.Code)
		}
		fields := map[string]string{
			"company_name": "???? ?????", "legal_entity_type": "limited_liability", "company_national_id": "14" + phone[2:], "registration_number": "123456", "economic_code": "123456789012",
			"province": string(enums.ProvinceTehran), "city": string(enums.CityTehran), "address": "?????? ?????", "postal_code": "1234567890",
			"representative_first_name": "???", "representative_last_name": "?????", "representative_position": "ceo", "representative_gender": "male", "representative_national_code": national,
			"phone": phone, "password": "EmployerTest123!",
		}

		var multipartBody bytes.Buffer
		writer := multipart.NewWriter(&multipartBody)
		for key, value := range fields {
			if err := writer.WriteField(key, value); err != nil {
				return err
			}
		}
		writer.WriteField("activities", string(enums.InterestProgramming))
		writer.WriteField("activities", string(enums.InterestDesign))
		establishment, err := writer.CreateFormFile("establishment_document", "gazette.pdf")
		if err != nil {
			return err
		}
		establishment.Write([]byte("%PDF-1.7\nnotice"))
		identity, err := writer.CreateFormFile("identity_document", "id.pdf")
		if err != nil {
			return err
		}
		identity.Write([]byte("%PDF-1.7\nidentity"))
		movie, err := writer.CreateFormFile("verification_video", "identity.mp4")
		if err != nil {
			return err
		}
		movie.Write(video)
		writer.Close()
		request, _ := http.NewRequest(http.MethodPost, "/v1/legal-employers/signup", bytes.NewReader(multipartBody.Bytes()))
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != 202 {
			t.Fatalf("signup returned %d: %s", response.Code, response.Body.String())
		}
		if strings.Contains(response.Body.String(), "password") || strings.Contains(response.Body.String(), "%PDF") {
			t.Fatal("private data leaked in signup response")
		}
		if sms.code == "" {
			t.Fatal("signup did not automatically send OTP")
		}
		pending, err := uow.Factory().LegalEmployerSignupRepository().GetSignup(context.Background(), phone)
		if err != nil {
			return err
		}
		if pending == nil || pending.Registration == nil || !passwords.Matches(pending.Registration.PasswordHash, fields["password"]) || !pending.Registration.PhoneVerifiedAt.IsZero() {
			t.Fatal("pending employer did not survive database round trip")
		}
		if account, err := uow.Factory().EmployerRepository().FindLegalByPhone(context.Background(), phone); err != nil || account != nil {
			t.Fatal("employer account created before verification")
		}
		loginBefore, _ := json.Marshal(general.LoginRequest{Phone: phone, Password: fields["password"]})
		if response := callAPI(router, http.MethodPost, "/v1/legal-employers/login", string(loginBefore), ""); response.Code != 401 {
			t.Fatal("pending employer could login")
		}
		wrongBody, _ := json.Marshal(general.VerifyOTPRequest{Phone: phone, OTP: "wrong"})
		if response := callAPI(router, http.MethodPost, "/v1/legal-employers/verify-otp", string(wrongBody), ""); response.Code != 401 {
			t.Fatal("wrong code accepted")
		}
		pending, err = uow.Factory().LegalEmployerSignupRepository().GetSignup(context.Background(), phone)
		if err != nil {
			return err
		}
		if pending.Challenge.Attempts != 1 {
			t.Fatal("failed verification attempt not persisted")
		}
		verifyBody, _ := json.Marshal(general.VerifyOTPRequest{Phone: phone, OTP: sms.code})
		if response := callAPI(router, http.MethodPost, "/v1/legal-employers/verify-otp", string(verifyBody), ""); response.Code != 201 {
			t.Fatalf("verification returned %d: %s", response.Code, response.Body.String())
		}
		if response := callAPI(router, http.MethodPost, "/v1/legal-employers/verify-otp", string(verifyBody), ""); response.Code != 401 {
			t.Fatal("OTP replay accepted")
		}
		e, err := uow.Factory().EmployerRepository().FindLegalByPhone(context.Background(), phone)
		if err != nil {
			return err
		}
		if e == nil || e.Employer.Type != enums.EmployerLegal || e.ReviewStatus != enums.EmployerPendingReview || e.ActivityLicense != nil || e.AuthorizationDocument != nil || e.Website != nil || len(e.Activities) != 2 {
			t.Fatal("parent/profile/nullable fields not preserved")
		}
		if !passwords.Matches(e.PasswordHash, fields["password"]) || len(storage.objects) != 3 {
			t.Fatal("password or documents not stored correctly")
		}
		var parent models.EmployerModel
		if err := tx.Where("id = ?", e.Employer.ID).Take(&parent).Error; err != nil {
			return err
		}
		if parent.Type != enums.EmployerLegal {
			t.Fatal("wrong shared employer type")
		}
		var nullLicenses int64
		tx.Model(&models.LegalEmployerModel{}).Where("employer_id = ? AND activity_license IS NULL AND authorization_document IS NULL", e.Employer.ID).Count(&nullLicenses)
		if nullLicenses != 1 {
			t.Fatal("license must be SQL NULL")
		}
		if !tx.Migrator().HasTable(&models.LegalEmployerModel{}) {
			t.Fatal("legal employer table missing")
		}
		if otp, err := uow.Factory().LegalEmployerSignupRepository().GetSignup(context.Background(), phone); err != nil || otp != nil {
			t.Fatal("OTP not consumed")
		}
		loginBody, _ := json.Marshal(general.LoginRequest{Phone: phone, Password: fields["password"]})
		login := callAPI(router, http.MethodPost, "/v1/legal-employers/login", string(loginBody), "")
		if login.Code != 200 {
			t.Fatalf("login returned %d", login.Code)
		}
		var credentials struct {
			Token string `json:"access_token"`
		}
		if err := json.Unmarshal(login.Body.Bytes(), &credentials); err != nil {
			return err
		}
		authorization := "Bearer " + credentials.Token
		if r := callAPI(router, http.MethodGet, "/v1/legal-employers/me", "", authorization); r.Code != 200 {
			t.Fatalf("employer me returned %d", r.Code)
		}
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", authorization); r.Code != 401 {
			t.Fatal("employer JWT accessed job seeker route")
		}
		individualToken, err := tokens.IssueAccessToken(e.Employer.ID, enums.RoleIndividualEmployer)
		if err != nil {
			return err
		}
		if r := callAPI(router, http.MethodGet, "/v1/legal-employers/me", "", "Bearer "+individualToken.Value); r.Code != 401 {
			t.Fatal("individual employer role accessed legal employer")
		}
		if r := callAPI(router, http.MethodGet, "/v1/individual-employers/me", "", authorization); r.Code != 401 {
			t.Fatal("legal employer role accessed individual employer")
		}
		jobToken, err := tokens.IssueAccessToken(e.Employer.ID, enums.RoleJobSeeker)
		if err != nil {
			return err
		}
		if _, err := employers.Authenticate(context.Background(), jobToken.Value); !errors.Is(err, domainerrors.ErrUnauthenticated) {
			t.Fatal("job seeker JWT accessed employer identity")
		}
		if r := callAPI(router, http.MethodGet, "/v1/legal-employers/documents/identity", "", authorization); r.Code != 200 {
			t.Fatal("private document download failed")
		}
		if r := callAPI(router, http.MethodPost, "/v1/legal-employers/logout", "", authorization); r.Code != 204 {
			t.Fatal("logout failed")
		}
		if r := callAPI(router, http.MethodGet, "/v1/legal-employers/me", "", authorization); r.Code != 401 {
			t.Fatal("revoked JWT accepted")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
