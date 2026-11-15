package persistence_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
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
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/media"
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
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed")
	}
	videoPath := filepath.Join(t.TempDir(), "verification.mp4")
	command := exec.Command("ffmpeg", "-v", "error", "-f", "lavfi", "-i", "color=c=black:s=16x16:r=1", "-t", "2", "-c:v", "mpeg4", "-y", videoPath)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("video fixture: %v %s", err, output)
	}
	video, err := os.ReadFile(videoPath)
	if err != nil {
		t.Fatal(err)
	}
	db := persistence.NewPostgresDatabase()
	rollback := errors.New("rollback test fixtures")
	err = db.Transaction(func(tx *gorm.DB) error {
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
		storage := &employerTestStorage{objects: map[string][]byte{}}
		signup := service.NewJobSeekerService(uow, sms, hasher, storage, media.NewValidator())
		auth := service.NewJobSeekerAuthService(uow, hasher, tokens)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		controllers := &wire.Controllers{GeneralControllers: &wire.GeneralControllers{
			JobSeekerController:     general.NewJobSeekerController(signup),
			JobSeekerAuthController: general.NewJobSeekerAuthController(auth),
		}}
		routes.SetUpRoutes(router, &wire.Application{Controllers: controllers, JobSeekerAuthService: auth})
		ctx := context.Background()
		phone, nationalCode, err := fixtureIdentity()
		if err != nil {
			return err
		}
		in := usecase.SignupJobSeeker{FirstName: "Integration", LastName: "Test", NationalCode: nationalCode, City: enums.CityTehran, Address: "Test address", BirthDate: valueobjects.JalaliDate{Year: 1388, Month: 1, Day: 1}, Gender: enums.GenderMale, Interests: []enums.Interest{enums.InterestProgramming}, Phone: phone, Password: "integration-password"}
		var signupBody bytes.Buffer
		writer := multipart.NewWriter(&signupBody)
		fields := map[string]string{"first_name": in.FirstName, "last_name": in.LastName, "national_code": in.NationalCode, "city": string(in.City), "address": in.Address,
			"birth_year": "1388", "birth_month": "1", "birth_day": "1", "gender": string(in.Gender), "interests": string(in.Interests[0]), "phone": in.Phone, "password": in.Password}
		for name, value := range fields {
			if err := writer.WriteField(name, value); err != nil {
				return err
			}
		}
		movie, err := writer.CreateFormFile("verification_video", "verification.mp4")
		if err != nil {
			return err
		}
		if _, err := movie.Write(video); err != nil {
			return err
		}
		if err := writer.Close(); err != nil {
			return err
		}
		request := httptest.NewRequest(http.MethodPost, "/v1/job-seekers/signup", &signupBody)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusAccepted {
			t.Fatalf("HTTP signup returned %d: %s", response.Code, response.Body.String())
		}
		repo := uow.Factory().SignupOTPRepository()
		pending, err := repo.GetOTP(ctx, in.Phone)
		if err != nil {
			return err
		}
		if pending.Registration == nil || !hasher.Matches(pending.Registration.PasswordHash, in.Password) {
			t.Fatal("pending JSONB profile/hash did not survive database round trip")
		}
		if pending.Registration.VerificationVideo == nil || !bytes.Equal(storage.objects[pending.Registration.VerificationVideo.Key], video) {
			t.Fatal("pending video reference did not survive JSONB round trip")
		}
		if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: in.Phone, Password: in.Password}); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
			t.Fatalf("pending account logged in: %v", err)
		}
		if _, err := signup.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: "wrong"}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
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
		verificationBody, err := json.Marshal(general.VerifyOTPRequest{Phone: in.Phone, OTP: sms.code})
		if err != nil {
			return err
		}
		if response := callAPI(router, http.MethodPost, "/v1/job-seekers/verify-otp", string(verificationBody), ""); response.Code != http.StatusCreated {
			t.Fatalf("HTTP verification returned %d", response.Code)
		}
		seeker, err := uow.Factory().JobSeekerRepository().FindByPhone(ctx, in.Phone)
		if err != nil {
			return err
		}
		if seeker == nil {
			t.Fatal("HTTP verification did not create the account")
		}
		if seeker.VerificationVideo == nil || seeker.VerificationVideo.Key != pending.Registration.VerificationVideo.Key {
			t.Fatal("video reference was not preserved when creating the account")
		}
		if seeker.PhoneVerifiedAt.IsZero() || seeker.BirthDate != in.BirthDate {
			t.Fatal("invalid persisted profile")
		}
		if _, err := signup.VerifyOTP(ctx, verify); !errors.Is(err, domainerrors.ErrInvalidOTP) {
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
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/documents/video", "", "Bearer "+login.AccessToken.Value); r.Code != 200 || r.Header().Get("Cache-Control") != "no-store" || !strings.Contains(r.Body.String(), "https://private.example/signed") {
			t.Fatalf("private video download failed: %d %s", r.Code, r.Body.String())
		}
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/documents/video", "", ""); r.Code != 401 {
			t.Fatal("anonymous video download accepted")
		}
		loginBody, err := json.Marshal(general.LoginRequest{Phone: in.Phone, Password: in.Password})
		if err != nil {
			return err
		}
		response = callAPI(router, http.MethodPost, "/v1/job-seekers/login", string(loginBody), "")
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
		if _, err := otherAuth.Authenticate(ctx, credentials.AccessToken); !errors.Is(err, domainerrors.ErrUnauthenticated) {
			t.Fatal("revocation did not survive a new service/repository instance")
		}
		newPhone, _, err := fixtureIdentity()
		if err != nil {
			return err
		}
		changeBody := fmt.Sprintf(`{"new_phone":%q}`, newPhone)
		bearer := "Bearer " + login.AccessToken.Value
		for _, endpoint := range []struct{ method, path, body string }{
			{http.MethodPost, "/v1/job-seekers/phone-change", changeBody},
			{http.MethodPost, "/v1/job-seekers/phone-change/verify", fmt.Sprintf(`{"new_phone":%q,"otp":"123456"}`, newPhone)},
			{http.MethodDelete, "/v1/job-seekers/me", `{"password":"integration-password"}`},
		} {
			if r := callAPI(router, endpoint.method, endpoint.path, endpoint.body, ""); r.Code != 401 {
				t.Fatalf("anonymous account change accepted: %s %s", endpoint.method, endpoint.path)
			}
		}
		if r := callAPI(router, http.MethodPost, "/v1/job-seekers/phone-change", changeBody, bearer); r.Code != 202 {
			t.Fatalf("request phone change: %d %s", r.Code, r.Body.String())
		}
		accountRepo := uow.Factory().JobSeekerRepository()
		challenge, err := accountRepo.GetPhoneChangeOTP(ctx, seeker.ID)
		if err != nil {
			return err
		}
		if challenge == nil || challenge.Phone != newPhone {
			t.Fatal("phone-change challenge not persisted")
		}
		wrongBody := fmt.Sprintf(`{"new_phone":%q,"otp":"wrong"}`, newPhone)
		if r := callAPI(router, http.MethodPost, "/v1/job-seekers/phone-change/verify", wrongBody, bearer); r.Code != 401 {
			t.Fatal("wrong phone-change OTP accepted")
		}
		challenge, err = accountRepo.GetPhoneChangeOTP(ctx, seeker.ID)
		if err != nil {
			return err
		}
		if challenge.Attempts != 1 {
			t.Fatal("phone-change failure did not commit")
		}
		verifyBody := fmt.Sprintf(`{"new_phone":%q,"otp":%q}`, newPhone, sms.code)
		if r := callAPI(router, http.MethodPost, "/v1/job-seekers/phone-change/verify", verifyBody, bearer); r.Code != 200 {
			t.Fatalf("verify phone change: %d %s", r.Code, r.Body.String())
		}
		updated, err := accountRepo.FindByID(ctx, seeker.ID)
		if err != nil {
			return err
		}
		if updated.Phone != newPhone {
			t.Fatal("phone update did not persist")
		}
		if r := callAPI(router, http.MethodPost, "/v1/job-seekers/phone-change/verify", verifyBody, bearer); r.Code != 401 {
			t.Fatal("phone-change OTP replay accepted")
		}
		if _, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: in.Phone, Password: in.Password}); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
			t.Fatal("old number still logs in")
		}
		newLogin, err := auth.Login(ctx, usecase.LoginJobSeeker{Phone: newPhone, Password: in.Password})
		if err != nil {
			return err
		}
		if r := callAPI(router, http.MethodDelete, "/v1/job-seekers/me", `{"password":"wrong-password"}`, bearer); r.Code != 401 {
			t.Fatal("wrong deletion password accepted")
		}
		if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", bearer); r.Code != 200 {
			t.Fatal("wrong deletion password removed account")
		}
		deleteBody := fmt.Sprintf(`{"password":%q}`, in.Password)
		if r := callAPI(router, http.MethodDelete, "/v1/job-seekers/me", deleteBody, bearer); r.Code != 200 {
			t.Fatalf("delete account: %d %s", r.Code, r.Body.String())
		}
		for _, token := range []string{login.AccessToken.Value, newLogin.AccessToken.Value} {
			if r := callAPI(router, http.MethodGet, "/v1/job-seekers/me", "", "Bearer "+token); r.Code != 401 {
				t.Fatal("deleted account session accepted")
			}
		}
		if _, ok := storage.objects[seeker.VerificationVideo.Key]; ok {
			t.Fatal("deleted account video was retained")
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
