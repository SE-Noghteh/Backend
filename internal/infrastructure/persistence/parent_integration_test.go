package persistence_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"
	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

// File inspection is tested separately; this test exercises real PostgreSQL,
// bcrypt, JWT, HTTP multipart binding, and transaction rollback with fake SMS/storage.
type parentIntegrationMedia struct{}

func (parentIntegrationMedia) Document(_ context.Context, f ports.UploadFile) (*ports.ValidatedFile, error) {
	size, err := f.Body.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}
	if _, err = f.Body.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	return &ports.ValidatedFile{UploadFile: f, ContentType: "image/png", Size: size, Extension: ".png"}, nil
}
func (parentIntegrationMedia) Video(ctx context.Context, f ports.UploadFile) (*ports.ValidatedFile, error) {
	v, err := (parentIntegrationMedia{}).Document(ctx, f)
	if err != nil {
		return nil, err
	}
	v.ContentType, v.Extension = "video/mp4", ".mp4"
	return v, nil
}

func TestPostgresParentSignupAndSharedParent(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	db := persistence.NewPostgresDatabase()
	rollback := errors.New("rollback parent fixtures")
	err := db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		uow := persistence.NewUnitOfWork(tx)
		hasher, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("parent-integration-key-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		sms := &capturedSMS{}
		storage := &employerTestStorage{objects: map[string][]byte{}}
		parentService := service.NewParentService(uow, sms, hasher, storage, parentIntegrationMedia{})
		seekerService := service.NewJobSeekerService(uow, sms, hasher, storage, parentIntegrationMedia{})
		auth := service.NewJobSeekerAuthService(uow, hasher, tokens)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		routes.SetUpRoutes(router, &wire.Application{JobSeekerAuthService: auth, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{ParentController: general.NewParentController(parentService), JobSeekerController: general.NewJobSeekerController(seekerService), JobSeekerAuthController: general.NewJobSeekerAuthController(auth)}}})
		newChild := func() (*entities.JobSeeker, string, error) {
			phone, national, err := fixtureIdentity()
			if err != nil {
				return nil, "", err
			}
			hash, err := hasher.Hash("child-password")
			if err != nil {
				return nil, "", err
			}
			child := &entities.JobSeeker{FirstName: "Child", LastName: "Test", NationalCode: national, City: enums.CityTehran, Address: "Test address", BirthDate: valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 16, Month: 1, Day: 1}, Gender: enums.GenderMale, Interests: []enums.Interest{enums.InterestProgramming}, Phone: phone, PasswordHash: hash, PhoneVerifiedAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), SignupStatus: entities.SignupFirstStageCompleted}
			if err := uow.Factory().JobSeekerRepository().Create(ctx, child); err != nil {
				return nil, "", err
			}
			token, err := tokens.IssueAccessToken(child.ID, enums.RoleJobSeeker)
			if err != nil {
				return nil, "", err
			}
			return child, "Bearer " + token.Value, nil
		}
		child, bearer, err := newChild()
		if err != nil {
			return err
		}
		if r := callAPI(router, "POST", "/v1/job-seekers/parent-approval", `{"skip":true}`, bearer); r.Code != 400 {
			t.Fatal("minor skipped stage two")
		}
		parentPhone, _, err := fixtureIdentity()
		if err != nil {
			return err
		}
		stageBody := fmt.Sprintf(`{"first_name":"Saved","last_name":"Parent","phone":%q}`, parentPhone)
		r := callAPI(router, "POST", "/v1/job-seekers/parent-approval", stageBody, bearer)
		if r.Code != 200 {
			t.Fatalf("parent invitation: %d %s", r.Code, r.Body.String())
		}
		var invitation usecase.ParentStageResult
		if err := json.Unmarshal(r.Body.Bytes(), &invitation); err != nil {
			return err
		}
		if invitation.InvitationCode == "" || invitation.SignupStatus != entities.SignupFirstStageCompleted {
			t.Fatal("missing invitation or incorrect stage")
		}
		codeBody := fmt.Sprintf(`{"code":%q}`, invitation.InvitationCode)
		if r := callAPI(router, "POST", "/v1/parents/invitations/validate", codeBody, ""); r.Code != 200 {
			t.Fatalf("invitation validation: %d %s", r.Code, r.Body.String())
		}
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		for k, v := range map[string]string{"code": invitation.InvitationCode, "relationship": "mother", "password": "parent-password", "consent_accepted": "true", "first_name": "Untrusted", "phone": "09120000000"} {
			if err := writer.WriteField(k, v); err != nil {
				return err
			}
		}
		for _, file := range []struct{ field, name, data string }{{"identity_document", "identity.png", "identity-image"}, {"consent_video", "consent.mp4", "consent-video"}} {
			part, err := writer.CreateFormFile(file.field, file.name)
			if err != nil {
				return err
			}
			if _, err := io.WriteString(part, file.data); err != nil {
				return err
			}
		}
		if err := writer.Close(); err != nil {
			return err
		}
		request := httptest.NewRequest("POST", "/v1/parents/signup", &body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != 202 {
			t.Fatalf("parent signup: %d %s", response.Code, response.Body.String())
		}
		parents := uow.Factory().ParentRepository()
		pending, err := parents.GetPending(ctx, parentPhone)
		if err != nil {
			return err
		}
		if pending == nil || !hasher.Matches(pending.Registration.PasswordHash, "parent-password") || pending.Registration.FirstName != "Saved" || pending.Registration.Phone != parentPhone || pending.Registration.IdentityDocument == nil || pending.Registration.ConsentVideo == nil {
			t.Fatal("pending parent profile did not survive JSONB round trip")
		}
		if p, err := parents.FindByPhone(ctx, parentPhone); err != nil || p != nil {
			t.Fatal("parent exists before OTP verification", err)
		}
		wrongBody := fmt.Sprintf(`{"code":%q,"otp":"bad"}`, invitation.InvitationCode)
		if r := callAPI(router, "POST", "/v1/parents/verify-otp", wrongBody, ""); r.Code != 401 {
			t.Fatal("invalid parent OTP accepted")
		}
		pending, err = parents.GetPending(ctx, parentPhone)
		if err != nil {
			return err
		}
		if pending.Challenge.Attempts != 1 {
			t.Fatal("failed parent OTP attempt rolled back")
		}
		verifyBody := fmt.Sprintf(`{"code":%q,"otp":%q}`, invitation.InvitationCode, sms.code)
		if r := callAPI(router, "POST", "/v1/parents/verify-otp", verifyBody, ""); r.Code != 201 {
			t.Fatalf("parent verification: %d %s", r.Code, r.Body.String())
		}
		parent, err := parents.FindByPhone(ctx, parentPhone)
		if err != nil {
			return err
		}
		linked, err := uow.Factory().JobSeekerRepository().FindByID(ctx, child.ID)
		if err != nil {
			return err
		}
		if parent == nil || linked.ParentID == nil || *linked.ParentID != parent.ID || linked.SignupStatus != entities.SignupSecondStageCompleted || linked.ParentRelationship != "mother" {
			t.Fatal("parent and stage were not linked atomically")
		}
		if r := callAPI(router, "POST", "/v1/parents/invitations/validate", codeBody, ""); r.Code != 400 {
			t.Fatal("consumed parent invitation accepted")
		}
		second, bearer2, err := newChild()
		if err != nil {
			return err
		}
		if r := callAPI(router, "POST", "/v1/job-seekers/parent-approval", stageBody, bearer2); r.Code != 200 {
			t.Fatalf("reuse parent: %d %s", r.Code, r.Body.String())
		}
		linked2, err := uow.Factory().JobSeekerRepository().FindByID(ctx, second.ID)
		if err != nil {
			return err
		}
		if linked2.ParentID == nil || *linked2.ParentID != parent.ID || linked2.SignupStatus != entities.SignupSecondStageCompleted {
			t.Fatal("second child did not share the same parent")
		}
		if r := callAPI(router, "DELETE", "/v1/job-seekers/me", `{"password":"child-password"}`, bearer); r.Code != 200 {
			t.Fatalf("delete first child: %d %s", r.Code, r.Body.String())
		}
		if p, err := parents.FindByPhone(ctx, parentPhone); err != nil || p == nil || len(storage.objects) != 2 {
			t.Fatal("deleting child removed shared parent or its documents", err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
