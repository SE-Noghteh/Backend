package persistence_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

func TestPostgresAdminEmployerReview(t *testing.T) {
	if os.Getenv("RUN_POSTGRES_INTEGRATION") != "1" {
		t.Skip("set RUN_POSTGRES_INTEGRATION=1")
	}
	if err := godotenv.Load("../../../.env"); err != nil {
		t.Fatal(err)
	}
	rollback := errors.New("rollback admin employer fixtures")
	err := persistence.NewPostgresDatabase().Transaction(func(tx *gorm.DB) error {
		ctx := context.Background()
		uow := persistence.NewUnitOfWork(tx)
		hasher, err := security.NewBcryptPasswordHasher()
		if err != nil {
			return err
		}
		tokens, err := security.NewJWTManager("admin-employer-review-test-secret-at-least-32-bytes", "test", "test", time.Hour)
		if err != nil {
			return err
		}
		phone, national, err := fixtureIdentity()
		if err != nil {
			return err
		}
		username := "test-admin-" + phone
		passwordHash, err := hasher.Hash("EmployerTest123!")
		if err != nil {
			return err
		}
		if err := tx.Exec("INSERT INTO admins (username, password) VALUES (?, ?)", username, passwordHash).Error; err != nil {
			return err
		}
		adminToken, err := tokens.IssueAdminAccessToken(username)
		if err != nil {
			return err
		}
		adminBearer := "Bearer " + adminToken.Value

		doc := valueobjects.StoredObject{Key: "private-key/identity.png", ContentType: "image/png", Size: 10}
		storage := &employerTestStorage{objects: map[string][]byte{doc.Key: []byte("media")}}
		employers := uow.Factory().EmployerRepository()
		now := time.Now().UTC()
		individual := &entities.IndividualEmployer{
			Phone: phone, NationalCode: national, PasswordHash: passwordHash, PhoneVerifiedAt: now, CreatedAt: now,
			FirstName: "Individual", LastName: "Employer", Gender: enums.GenderMale,
			BirthDate:        valueobjects.JalaliDate{Year: 1375, Month: 1, Day: 1},
			IdentityDocument: doc, VerificationVideo: doc, Province: enums.ProvinceTehran, City: enums.CityTehran,
		}
		if err := employers.CreateIndividual(ctx, individual); err != nil {
			return err
		}
		legal := &entities.LegalEmployer{
			Phone: phone, CompanyNationalID: "1" + national, PasswordHash: passwordHash, PhoneVerifiedAt: now, CreatedAt: now,
			CompanyName: "Legal Employer", LegalEntityType: "limited_liability", RepresentativePosition: "ceo", RepresentativeGender: enums.GenderMale,
			Activities: []enums.Interest{enums.InterestDesign}, EstablishmentDocument: doc, IdentityDocument: doc,
			VerificationVideo: doc, AuthorizationDocument: &doc, Province: enums.ProvinceTehran, City: enums.CityTehran,
		}
		if err := employers.CreateLegal(ctx, legal); err != nil {
			return err
		}
		individualService := service.NewIndividualEmployerService(uow, nil, hasher, tokens, storage, nil)
		legalService := service.NewLegalEmployerService(uow, nil, hasher, tokens, storage, nil)
		admin := service.NewAdminService(uow, hasher, tokens, storage)
		gin.SetMode(gin.TestMode)
		router := gin.New()
		routes.SetUpRoutes(router, &wire.Application{AdminService: admin, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{
			AdminController:              general.NewAdminController(admin),
			IndividualEmployerController: general.NewIndividualEmployerController(individualService),
			LegalEmployerController:      general.NewLegalEmployerController(legalService),
			EmployerJobPostingController: general.NewEmployerJobPostingController(service.NewEmployerJobPostingService(uow, individualService, legalService)),
		}}})

		const path = "/v1/admin/employers"
		individualPath := fmt.Sprintf("%s/%d", path, individual.Employer.ID)
		legalPath := fmt.Sprintf("%s/%d", path, legal.Employer.ID)
		list := func(query string) general.AdminEmployersResponse {
			response := callAPI(router, "GET", path+query, "", adminBearer)
			var result general.AdminEmployersResponse
			if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &result) != nil {
				t.Fatalf("list %s: %d %s", query, response.Code, response.Body.String())
			}
			return result
		}
		if result := list(""); result.Total != 2 || len(result.Data) != 2 || result.Data[0].Name != "Individual Employer" || result.Data[1].Name != "Legal Employer" {
			t.Fatalf("unexpected default employer list: %+v", result)
		}
		if result := list("?type=legal&limit=1"); result.Total != 1 || len(result.Data) != 1 || result.Data[0].ID != legal.Employer.ID {
			t.Fatalf("legal filter failed: %+v", result)
		}
		if result := list("?status=all&limit=1&offset=1"); result.Total != 2 || len(result.Data) != 1 || result.Data[0].ID != legal.Employer.ID {
			t.Fatalf("pagination failed: %+v", result)
		}
		for _, invalid := range []string{"?type=parent", "?status=unknown", "?limit=101", "?offset=-1"} {
			if r := callAPI(router, "GET", path+invalid, "", adminBearer); r.Code != 400 {
				t.Fatalf("invalid filter %s returned %d", invalid, r.Code)
			}
		}
		for _, fixture := range []struct {
			path     string
			kind     enums.EmployerType
			document string
			missing  string
		}{
			{individualPath, enums.EmployerIndividual, "identity", "business-license"},
			{legalPath, enums.EmployerLegal, "authorization", "activity-license"},
		} {
			r := callAPI(router, "GET", fixture.path, "", adminBearer)
			var detail general.AdminEmployerResponse
			if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &detail) != nil || detail.EmployerType != fixture.kind || !slices.Contains(detail.AvailableDocuments, fixture.document) || slices.Contains(detail.AvailableDocuments, fixture.missing) {
				t.Fatalf("unexpected detail: %d %s", r.Code, r.Body.String())
			}
			if (detail.Individual != nil) != (fixture.kind == enums.EmployerIndividual) || (detail.Legal != nil) != (fixture.kind == enums.EmployerLegal) {
				t.Fatal("unexpected employer profile type in response")
			}
			if strings.Contains(r.Body.String(), doc.Key) || strings.Contains(r.Body.String(), passwordHash) {
				t.Fatal("admin detail exposed a storage key or password hash")
			}
			r = callAPI(router, "GET", fixture.path+"/documents/"+fixture.document, "", adminBearer)
			if r.Code != 200 || r.Header().Get("Cache-Control") != "no-store" || !strings.Contains(r.Body.String(), "https://private.example/signed") {
				t.Fatalf("document download failed: %d %s", r.Code, r.Body.String())
			}
			if r := callAPI(router, "GET", fixture.path+"/documents/"+fixture.missing, "", adminBearer); r.Code != 404 {
				t.Fatalf("missing document returned %d", r.Code)
			}
		}
		if r := callAPI(router, "GET", individualPath+"/documents/establishment", "", adminBearer); r.Code != 400 {
			t.Fatalf("wrong employer document kind returned %d", r.Code)
		}
		if r := callAPI(router, "GET", path, "", ""); r.Code != 401 {
			t.Fatal("employer list accepted anonymous request")
		}
		if r := callAPI(router, "GET", path+"/9223372036854775807", "", adminBearer); r.Code != 404 {
			t.Fatal("missing employer returned an unexpected status")
		}

		const postingBody = `{"title":"Review test","description":"Draft","details":{"opportunity_type":"work","category":"Design","work_mode":"online","minimum_age":18,"maximum_age":22,"apply_deadline":"2028-02-01","delivery_deadline":"2028-02-15","budget_toman":100000}}`
		for _, fixture := range []struct {
			id    uint64
			route string
			role  enums.AccountRole
		}{
			{individual.Employer.ID, "individual-employers", enums.RoleIndividualEmployer},
			{legal.Employer.ID, "legal-employers", enums.RoleLegalEmployer},
		} {
			reviewPath := fmt.Sprintf("%s/%d/review", path, fixture.id)
			token, err := tokens.IssueEmployerAccessToken(fixture.id, fixture.role, 0)
			if err != nil {
				return err
			}
			employerBearer := "Bearer " + token.Value
			const postingPath = "/v1/employer/job-postings"
			if r := callAPI(router, "POST", reviewPath, `{"approved":true}`, employerBearer); r.Code != 401 {
				t.Fatal("employer credentials were accepted for admin review")
			}
			if r := callAPI(router, "POST", postingPath, postingBody, employerBearer); r.Code != 403 {
				t.Fatalf("pending employer posting returned %d", r.Code)
			}
			if r := callAPI(router, "POST", reviewPath, `{"approved":true}`, adminBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"review_status":"approved"`) {
				t.Fatalf("approval failed: %d %s", r.Code, r.Body.String())
			}
			if r := callAPI(router, "POST", postingPath, postingBody, employerBearer); r.Code != 201 {
				t.Fatalf("approved employer could not create a draft: %d %s", r.Code, r.Body.String())
			}
			if r := callAPI(router, "POST", reviewPath, `{"approved":false,"message":"Replace the uploaded identity image"}`, adminBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"review_status":"rejected"`) {
				t.Fatalf("rejection failed: %d %s", r.Code, r.Body.String())
			}
			r := callAPI(router, "GET", "/v1/"+fixture.route+"/me", "", employerBearer)
			if r.Code != 200 || !strings.Contains(r.Body.String(), `"admin_review_message":"Replace the uploaded identity image"`) || !strings.Contains(r.Body.String(), `"admin_reviewed_at"`) {
				t.Fatalf("employer did not receive review feedback: %d %s", r.Code, r.Body.String())
			}
			if r := callAPI(router, "POST", postingPath, postingBody, employerBearer); r.Code != 403 {
				t.Fatalf("rejected employer posting returned %d", r.Code)
			}
		}
		if result := list("?status=rejected"); result.Total != 2 || len(result.Data) != 2 {
			t.Fatalf("rejected employer filter failed: %+v", result)
		}
		persisted, err := employers.FindLegalByID(ctx, legal.Employer.ID)
		if err != nil {
			return err
		}
		if persisted.ReviewedBy != username || persisted.ReviewedAt == nil || persisted.ReviewStatus != enums.EmployerRejected {
			t.Fatal("review audit data did not persist")
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
}
