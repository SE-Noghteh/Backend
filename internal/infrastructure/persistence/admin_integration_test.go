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
	"zedlancer/backend/internal/infrastructure/persistence/models"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

func TestPostgresAdminLoginAndDependentAcceptance(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	db := persistence.NewPostgresDatabase()
	if db.Migrator().HasColumn("job_seekers", "signup_status") {
		t.Fatal("obsolete signup stage column remains")
	}
	rollback := errors.New("rollback admin fixtures")
	columns, err := db.Migrator().ColumnTypes(&models.AdminModel{})
	if err != nil {
		t.Fatal(err)
	}
	if len(columns) != 2 {
		t.Fatal("admin table must have only username and password")
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		uow := persistence.NewUnitOfWork(tx)
		phone, national, err := fixtureIdentity()
		if err != nil {
			return err
		}
		username := "test-admin-" + phone
		if err := tx.Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO admins (username, password) VALUES (?, crypt(?, gen_salt('bf', 10)))", username, "admin-test-password").Error; err != nil {
			return err
		}
		hasher, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("admin-integration-secret-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		storage := &employerTestStorage{objects: map[string][]byte{"job-seekers/test/video.mp4": []byte("video"), "parents/test/identity.png": []byte("image"), "parents/test/consent.mp4": []byte("consent")}}
		sms := &capturedSMS{}
		adminService := service.NewAdminService(uow, hasher, tokens, storage)
		seekerAuth := service.NewJobSeekerAuthService(uow, hasher, tokens)
		seekerService := service.NewJobSeekerService(uow, sms, hasher, storage, parentIntegrationMedia{})
		router := gin.New()
		gin.SetMode(gin.TestMode)
		routes.SetUpRoutes(router, &wire.Application{AdminService: adminService, JobSeekerAuthService: seekerAuth, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{AdminController: general.NewAdminController(adminService), JobSeekerController: general.NewJobSeekerController(seekerService), JobSeekerAuthController: general.NewJobSeekerAuthController(seekerAuth)}}})
		hash, err := hasher.Hash("child-password")
		if err != nil {
			return err
		}
		parentPhone, _, err := fixtureIdentity()
		if err != nil {
			return err
		}
		parent := &entities.Parent{FirstName: "Parent", LastName: "Test", Relationship: "mother", Phone: parentPhone, PasswordHash: hash, IdentityDocument: &valueobjects.StoredObject{Key: "parents/test/identity.png"}, ConsentVideo: &valueobjects.StoredObject{Key: "parents/test/consent.mp4"}, PhoneVerifiedAt: time.Now().UTC(), ConsentAcceptedAt: time.Now().UTC(), CreatedAt: time.Now().UTC()}
		if err := uow.Factory().ParentRepository().Create(ctx, parent); err != nil {
			return err
		}
		child := &entities.JobSeeker{FirstName: "Child", LastName: "Test", Phone: phone, NationalCode: national, PasswordHash: hash, City: enums.CityTehran, Address: "Test address", BirthDate: valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 16, Month: 1, Day: 1}, Gender: enums.GenderMale, Interests: []enums.Interest{enums.InterestProgramming}, PhoneVerifiedAt: time.Now().UTC(), CreatedAt: time.Now().UTC(), ParentID: &parent.ID, ParentRelationship: "mother", VerificationVideo: &valueobjects.StoredObject{Key: "job-seekers/test/video.mp4"}}
		if err := uow.Factory().JobSeekerRepository().Create(ctx, child); err != nil {
			return err
		}
		incomplete := *child
		incomplete.ID = 0
		incomplete.ParentID = nil
		incomplete.Phone, incomplete.NationalCode, err = fixtureIdentity()
		if err != nil {
			return err
		}
		if err := uow.Factory().JobSeekerRepository().Create(ctx, &incomplete); err != nil {
			return err
		}
		childLogin, err := seekerAuth.Login(ctx, usecase.LoginJobSeeker{Phone: phone, Password: "child-password"})
		if err != nil {
			return err
		}
		userBearer := "Bearer " + childLogin.AccessToken.Value
		loginBody := fmt.Sprintf(`{"username":%q,"password":"admin-test-password"}`, username)
		response := callAPI(router, "POST", "/v1/admin/login", loginBody, "")
		if response.Code != 200 {
			t.Fatalf("documented SQL credentials cannot log in: %d %s", response.Code, response.Body.String())
		}
		var login general.AdminLoginResponse
		if err := json.Unmarshal(response.Body.Bytes(), &login); err != nil {
			return err
		}
		bearer := "Bearer " + login.AccessToken
		if login.Username != username || login.AccessToken == "" || strings.Contains(response.Body.String(), "password") {
			t.Fatal("incorrect login response")
		}
		if r := callAPI(router, "POST", "/v1/admin/login", fmt.Sprintf(`{"username":%q,"password":"wrong-password"}`, username), ""); r.Code != 401 {
			t.Fatal("invalid password accepted")
		}
		for _, token := range []string{"", userBearer} {
			if r := callAPI(router, "GET", "/v1/admin/job-seekers", "", token); r.Code != 401 {
				t.Fatal("nonadmin accessed review list")
			}
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", bearer); r.Code != 401 {
			t.Fatal("admin token accessed job seeker account")
		}
		// Admin responses derive all child IDs from the current parent connections.
		if err := uow.Factory().ParentRepository().SetParentConnection(ctx, incomplete.ID, &parent.ID, "mother"); err != nil {
			return err
		}
		for _, path := range []string{"/v1/admin/parents", fmt.Sprintf("/v1/admin/parents/%d", parent.ID)} {
			r := callAPI(router, "GET", path, "", bearer)
			if r.Code != 200 {
				t.Fatalf("parent response failed: %d %s", r.Code, r.Body.String())
			}
			var result general.AdminParentResponse
			if path == "/v1/admin/parents" {
				var list general.AdminParentsResponse
				if err := json.Unmarshal(r.Body.Bytes(), &list); err != nil {
					return err
				}
				if len(list.Data) != 1 {
					t.Fatalf("unexpected parents: %s", r.Body.String())
				}
				result = list.Data[0]
			} else if err := json.Unmarshal(r.Body.Bytes(), &result); err != nil {
				return err
			}
			if len(result.ChildIDs) != 2 || result.ChildIDs[0] != child.ID || result.ChildIDs[1] != incomplete.ID {
				t.Fatalf("admin parent response missing connected children: %s", r.Body.String())
			}
		}
		if err := uow.Factory().ParentRepository().SetParentConnection(ctx, incomplete.ID, nil, ""); err != nil {
			return err
		}
		for _, endpoint := range []string{"/v1/admin/job-seekers", "/v1/admin/parents"} {
			r := callAPI(router, "GET", endpoint, "", bearer)
			if r.Code != 200 {
				t.Fatalf("list failed: %d %s", r.Code, r.Body.String())
			}
			var list struct {
				Total int64             `json:"total"`
				Data  []json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(r.Body.Bytes(), &list); err != nil {
				return err
			}
			if list.Total != 1 || len(list.Data) != 1 || strings.Contains(r.Body.String(), "password") || strings.Contains(r.Body.String(), "test/video.mp4") || strings.Contains(r.Body.String(), "test/identity.png") {
				t.Fatal("list contains incomplete accounts or sensitive data")
			}
		}
		for _, path := range []string{fmt.Sprintf("/v1/admin/job-seekers/%d/documents/video", child.ID), fmt.Sprintf("/v1/admin/parents/%d/documents/identity", parent.ID), fmt.Sprintf("/v1/admin/parents/%d/documents/consent-video", parent.ID)} {
			r := callAPI(router, "GET", path, "", bearer)
			if r.Code != 200 || r.Header().Get("Cache-Control") != "no-store" || !strings.Contains(r.Body.String(), "https://private.example/signed") {
				t.Fatalf("private document download failed: %d %s", r.Code, r.Body.String())
			}
		}
		childReviewPath := fmt.Sprintf("/v1/admin/job-seekers/%d/review", child.ID)
		if r := callAPI(router, "POST", childReviewPath, `{}`, bearer); r.Code != 400 {
			t.Fatal("missing decision accepted")
		}
		if r := callAPI(router, "POST", fmt.Sprintf("/v1/admin/job-seekers/%d/review", incomplete.ID), `{"approved":true}`, bearer); r.Code != 409 {
			t.Fatal("incomplete signup approved")
		}
		if r := callAPI(router, "POST", childReviewPath, `{"approved":true}`, bearer); r.Code != 200 {
			t.Fatalf("child approval failed: %d %s", r.Code, r.Body.String())
		}
		persisted, err := uow.Factory().JobSeekerRepository().FindByID(ctx, child.ID)
		if err != nil {
			return err
		}
		if persisted.AdminApprovalStatus != enums.AdminReviewApproved || persisted.AdminReviewedBy != username || persisted.AdminReviewedAt == nil {
			t.Fatal("review audit was not persisted")
		}
		persistedParent, err := uow.Factory().AdminReviewRepository().FindParent(ctx, parent.ID)
		if err != nil {
			return err
		}
		if persistedParent.AdminApprovalStatus != enums.AdminReviewPending {
			t.Fatal("child approval incorrectly approved the parent")
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", userBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"pending"`) {
			t.Fatal("minor must remain pending until parent is admin approved")
		}
		if r := callAPI(router, "GET", "/v1/admin/job-seekers", "", bearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"total":1`) {
			t.Fatal("minor waiting for parent approval was omitted from pending list")
		}
		if r := callAPI(router, "GET", "/v1/admin/job-seekers?status=approved", "", bearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"total":0`) {
			t.Fatal("accepted filter included a minor awaiting parent approval")
		}
		parentApprovalPath := fmt.Sprintf("/v1/admin/parents/%d/review", parent.ID)
		parentReview := callAPI(router, "POST", parentApprovalPath, `{"approved":true}`, bearer)
		if parentReview.Code != 200 {
			t.Fatal("parent approval failed")
		}
		var reviewedParent general.AdminParentReviewResponse
		if err := json.Unmarshal(parentReview.Body.Bytes(), &reviewedParent); err != nil {
			return err
		}
		if len(reviewedParent.Data.ChildIDs) != 1 || reviewedParent.Data.ChildIDs[0] != child.ID {
			t.Fatal("review response has stale child connections", parentReview.Body.String())
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", userBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"approved"`) || !strings.Contains(r.Body.String(), `"parent_connected":true`) || !strings.Contains(r.Body.String(), `"admin_approved":true`) || !strings.Contains(r.Body.String(), `"admin_approved_parent":true`) {
			t.Fatalf("minor not accepted after all three approvals: %s", r.Body.String())
		}
		if r := callAPI(router, "GET", "/v1/admin/job-seekers?status=pending", "", bearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"total":0`) {
			t.Fatal("accepted minor still listed as pending")
		}
		if r := callAPI(router, "GET", "/v1/admin/job-seekers?status=approved", "", bearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"total":1`) {
			t.Fatal("accepted minor omitted from accepted filter")
		}
		if r := callAPI(router, "POST", childReviewPath, `{"approved":false,"message":"Video is unclear"}`, bearer); r.Code != 200 || !strings.Contains(r.Body.String(), "Video is unclear") {
			t.Fatal("rejection failed")
		}
		for _, suffix := range []string{"", "?status=pending"} {
			r := callAPI(router, "GET", "/v1/admin/job-seekers"+suffix, "", bearer)
			if r.Code != 200 || !strings.Contains(r.Body.String(), `"total":0`) {
				t.Fatalf("rejected job seeker included in pending list %s: %s", suffix, r.Body.String())
			}
		}
		for _, status := range []string{"rejected", "all"} {
			r := callAPI(router, "GET", "/v1/admin/job-seekers?status="+status, "", bearer)
			if r.Code != 200 || !strings.Contains(r.Body.String(), `"total":1`) || !strings.Contains(r.Body.String(), phone) {
				t.Fatalf("rejected job seeker missing from %s: %s", status, r.Body.String())
			}
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", userBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"pending"`) || !strings.Contains(r.Body.String(), "Video is unclear") {
			t.Fatal("rejected user cannot see status/message")
		}
		if _, err := seekerAuth.Login(ctx, usecase.LoginJobSeeker{Phone: phone, Password: "child-password"}); err != nil {
			t.Fatal("rejected user login was blocked")
		}
		parentPath := fmt.Sprintf("/v1/admin/parents/%d/review", parent.ID)
		if r := callAPI(router, "POST", parentPath, `{"approved":false}`, bearer); r.Code != 200 || !strings.Contains(r.Body.String(), "Parent was not approved") {
			t.Fatal("optional rejection message failed")
		}
		for _, suffix := range []string{"", "?status=pending"} {
			r := callAPI(router, "GET", "/v1/admin/parents"+suffix, "", bearer)
			if r.Code != 200 || !strings.Contains(r.Body.String(), `"total":0`) {
				t.Fatalf("rejected parent included in pending list %s: %s", suffix, r.Body.String())
			}
		}
		for _, status := range []string{"rejected", "all"} {
			r := callAPI(router, "GET", "/v1/admin/parents?status="+status, "", bearer)
			if r.Code != 200 || !strings.Contains(r.Body.String(), `"total":1`) || !strings.Contains(r.Body.String(), parentPhone) {
				t.Fatalf("rejected parent missing from %s: %s", status, r.Body.String())
			}
		}
		if r := callAPI(router, "POST", parentPath, `{"approved":true}`, bearer); r.Code != 200 {
			t.Fatal("parent approval failed")
		}
		persisted, err = uow.Factory().JobSeekerRepository().FindByID(ctx, child.ID)
		if err != nil {
			return err
		}
		if persisted.AdminApprovalStatus != enums.AdminReviewRejected {
			t.Fatal("parent approval changed child rejection")
		}
		// Parent review changes are reflected by every linked minor without updating child rows.
		sibling := *child
		sibling.ID = 0
		sibling.AdminReview = valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewApproved}
		sibling.Phone, sibling.NationalCode, err = fixtureIdentity()
		if err != nil {
			return err
		}
		if err := uow.Factory().JobSeekerRepository().Create(ctx, &sibling); err != nil {
			return err
		}
		siblingToken, err := tokens.IssueAccessToken(sibling.ID, enums.RoleJobSeeker)
		if err != nil {
			return err
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", "Bearer "+siblingToken.Value); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"approved"`) {
			t.Fatalf("second child did not inherit parent approval: %s", r.Body.String())
		}
		if r := callAPI(router, "POST", parentPath, `{"approved":false}`, bearer); r.Code != 200 {
			t.Fatal("parent re-review failed")
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", "Bearer "+siblingToken.Value); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"pending"`) || !strings.Contains(r.Body.String(), `"admin_approved":true`) || !strings.Contains(r.Body.String(), `"admin_approved_parent":false`) {
			t.Fatalf("second child stayed accepted after parent rejection: %s", r.Body.String())
		}
		if r := callAPI(router, "POST", parentPath, `{"approved":true}`, bearer); r.Code != 200 {
			t.Fatal("parent re-approval failed")
		}
		if r := callAPI(router, "GET", "/v1/job-seekers/me", "", "Bearer "+siblingToken.Value); !strings.Contains(r.Body.String(), `"signup_status":"approved"`) {
			t.Fatal("parent re-approval did not restore minor acceptance")
		}
		// Adults do not need to connect or skip a parent stage to be reviewed and accepted.
		adult := *child
		adult.ID = 0
		adult.ParentID = nil
		adult.ParentRelationship = ""
		adult.AdminReview = valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewPending}
		adult.BirthDate = valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 20, Month: 1, Day: 1}
		adult.Phone, adult.NationalCode, err = fixtureIdentity()
		if err != nil {
			return err
		}
		if err := uow.Factory().JobSeekerRepository().Create(ctx, &adult); err != nil {
			return err
		}
		if r := callAPI(router, "GET", "/v1/admin/job-seekers?status=pending", "", bearer); !strings.Contains(r.Body.String(), adult.Phone) {
			t.Fatal("adult was hidden until optional parent stage")
		}
		if r := callAPI(router, "POST", fmt.Sprintf("/v1/admin/job-seekers/%d/review", adult.ID), `{"approved":true}`, bearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"signup_status":"approved"`) || !strings.Contains(r.Body.String(), `"parent_connected":false`) || !strings.Contains(r.Body.String(), `"admin_approved_parent":false`) {
			t.Fatalf("adult acceptance incorrectly required parent: %d %s", r.Code, r.Body.String())
		}
		for path, status := range map[string]int{"/v1/admin/job-seekers?limit=101": 400, "/v1/admin/parents?status=bad": 400, "/v1/admin/parents/0": 400, "/v1/admin/parents/99999999": 404} {
			if r := callAPI(router, "GET", path, "", bearer); r.Code != status {
				t.Fatalf("%s returned %d", path, r.Code)
			}
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
