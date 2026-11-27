package persistence_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/gorm"

	"zedlancer/backend/internal/application/service"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/media"
	"zedlancer/backend/internal/infrastructure/persistence"
	"zedlancer/backend/internal/infrastructure/persistence/models"
	"zedlancer/backend/internal/infrastructure/security"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

const internshipDraftBody = `{"title":"Content internship","details":{"category":"Content","work_mode":"online","minimum_age":14,"maximum_age":18,"accept_under_18":true,"requires_parent_approval":true,"apply_deadline":"2028-02-01","internship":{"learning_outcomes":"Plan content","responsibilities":"Write posts","start_date":"2028-02-15","duration_months":3,"hours_per_week":6,"issues_certificate":true,"has_monthly_stipend":true,"monthly_stipend_toman":500000}}}`

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
		routes.SetUpRoutes(router, &wire.Application{AdminService: admin, JobSeekerAuthService: service.NewJobSeekerAuthService(uow, hasher, tokens), Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{
			AdminController:              general.NewAdminController(admin),
			IndividualEmployerController: general.NewIndividualEmployerController(individualService),
			LegalEmployerController:      general.NewLegalEmployerController(legalService),
			EmployerJobPostingController: general.NewEmployerJobPostingController(service.NewEmployerJobPostingService(uow, individualService, legalService, storage, media.NewValidator())),
		}}})
		checkOpportunitySearch(t, tx, router, storage, individual.Employer.ID, legal.Employer.ID, phone, tokens)

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
			const internshipPath = "/v1/employer/internship-postings"
			checkPostingAccess := func(authorization string, status int) {
				t.Helper()
				for _, posting := range []struct{ path, body string }{
					{postingPath, postingBody}, {internshipPath, internshipDraftBody},
				} {
					if r := callAPI(router, "POST", posting.path, posting.body, authorization); r.Code != status {
						t.Fatalf("POST %s returned %d, want %d: %s", posting.path, r.Code, status, r.Body.String())
					}
				}
			}
			checkPostingAccess("", 401)
			checkPostingAccess(adminBearer, 401)
			checkPostingAccess(employerBearer, 403)
			if r := callAPI(router, "POST", reviewPath, `{"approved":true}`, adminBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"review_status":"approved"`) {
				t.Fatalf("approval failed: %d %s", r.Code, r.Body.String())
			}
			// An approved record with a missing phone verification must still be rejected,
			// even when the request carries an otherwise valid employer token.
			profileTable := strings.ReplaceAll(fixture.route, "-", "_")
			if err := tx.Table(profileTable).Where("employer_id = ?", fixture.id).
				UpdateColumn("phone_verified_at", time.Time{}).Error; err != nil {
				return err
			}
			checkPostingAccess(employerBearer, 401)
			if err := tx.Table(profileTable).Where("employer_id = ?", fixture.id).
				UpdateColumn("phone_verified_at", now).Error; err != nil {
				return err
			}
			r := callAPI(router, "POST", postingPath, postingBody, employerBearer)
			var draft general.JobPostingCreatedResponse
			if r.Code != 201 || json.Unmarshal(r.Body.Bytes(), &draft) != nil || draft.ID == 0 {
				t.Fatalf("approved employer could not create a draft: %d %s", r.Code, r.Body.String())
			}
			checkJobPostingReview(t, router, storage, draft.ID, postingBody, employerBearer)
			if r := callAPI(router, "POST", reviewPath, `{"approved":false,"message":"Replace the uploaded identity image"}`, adminBearer); r.Code != 200 || !strings.Contains(r.Body.String(), `"review_status":"rejected"`) {
				t.Fatalf("rejection failed: %d %s", r.Code, r.Body.String())
			}
			r = callAPI(router, "GET", "/v1/"+fixture.route+"/me", "", employerBearer)
			if r.Code != 200 || !strings.Contains(r.Body.String(), `"admin_review_message":"Replace the uploaded identity image"`) || !strings.Contains(r.Body.String(), `"admin_reviewed_at"`) {
				t.Fatalf("employer did not receive review feedback: %d %s", r.Code, r.Body.String())
			}
			checkPostingAccess(employerBearer, 403)
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

func checkOpportunitySearch(t *testing.T, tx *gorm.DB, router http.Handler, storage *employerTestStorage, individualID, legalID uint64, marker string, tokens *security.JWTManager) {
	t.Helper()
	now := time.Now()
	date := func(days int) string { return now.AddDate(0, 0, days).Format(time.DateOnly) }
	work := entities.JobPostingDetails{
		OpportunityType: entities.WorkPosting, Category: "طراحی گرافيك", SkillLevel: entities.PostingSkillBeginner,
		WorkMode: "online", MinimumAge: 14, MaximumAge: 17, AcceptUnder18: true, RequiresParentApproval: true,
		ApplyDeadline: date(10), DeliveryDeadline: date(12), BudgetToman: 2400000,
		Skills: []string{"طراحی پست"}, Deliverables: []string{"۱۰ پست"}, DeliveryFormats: []string{"PNG"},
	}
	internship := entities.JobPostingDetails{
		OpportunityType: entities.InternshipPosting, Category: "تولید محتوا", SkillLevel: entities.PostingSkillIntermediate,
		WorkMode: "onsite", MinimumAge: 18, MaximumAge: 25, ApplyDeadline: date(15),
		Internship: &entities.InternshipTerms{DurationMonths: 3},
		Skills:     []string{"طراحی پست", "تولید محتوا"},
	}
	seed := func(employerID uint64, title, description, status string, createdAt time.Time, details entities.JobPostingDetails) uint64 {
		t.Helper()
		encoded, err := json.Marshal(details)
		if err != nil {
			t.Fatal(err)
		}
		posting := models.JobPostingModel{
			EmployerID: employerID, Title: title, Description: description, Status: status, Details: encoded, CreatedAt: createdAt,
		}
		if err := tx.Create(&posting).Error; err != nil {
			t.Fatal(err)
		}
		return posting.ID
	}
	workID := seed(individualID, marker+" طراحي‌گرافیک", "Work summary", "published", now.Add(-time.Hour), work)
	internshipID := seed(legalID, marker+" کارآموزی", "Internship summary", "published", now, internship)
	draftID := seed(individualID, marker+" Draft", "", "draft", now, work)
	checkJobSeekerOpportunityDashboard(t, tx, router, tokens, marker, workID, internshipID, draftID)
	seed(individualID, "Different title", marker, "published", now, work)
	expired := work
	expired.ApplyDeadline = date(-2)
	expiredID := seed(individualID, marker+" Expired", "", "published", now, expired)
	publishedAt := now.Add(-30 * time.Minute)
	const attachmentKey = "job-postings/test-detail.pdf"
	if err := tx.Model(&models.JobPostingModel{}).Where("id = ?", workID).Updates(models.JobPostingModel{
		PublishedAt: &publishedAt, Attachments: []entities.JobPostingAttachment{
			{Name: "brief.pdf", Key: attachmentKey, ContentType: "application/pdf", Size: 10},
		},
	}).Error; err != nil {
		t.Fatal(err)
	}
	storage.objects[attachmentKey] = []byte("Test brief")
	defer delete(storage.objects, attachmentKey)
	for _, scenario := range []struct {
		id        uint64
		status    int
		accepting bool
	}{
		{workID, 200, true}, {internshipID, 200, true}, {expiredID, 200, false}, {draftID, 404, false}, {0, 400, false},
	} {
		r := callAPI(router, "GET", fmt.Sprintf("/v1/opportunities/%d", scenario.id), "", "")
		if r.Code != scenario.status {
			t.Fatalf("GET opportunity %d: %d %s", scenario.id, r.Code, r.Body.String())
		}
		if scenario.status != 200 {
			continue
		}
		var detail usecase.OpportunityDetail
		if err := json.Unmarshal(r.Body.Bytes(), &detail); err != nil {
			t.Fatal(err)
		}
		if detail.PublishedOpportunity == nil || detail.ID != scenario.id || detail.AcceptingApplications != scenario.accepting ||
			detail.Details == nil || detail.Attachments == nil || r.Header().Get("Cache-Control") != "no-store" ||
			strings.Contains(r.Body.String(), attachmentKey) {
			t.Fatalf("invalid public detail: %s", r.Body.String())
		}
		if scenario.id == workID && (detail.Description != "Work summary" || detail.PublishedAt == nil ||
			!slices.Equal(detail.Details.Deliverables, work.Deliverables) || !slices.Equal(detail.Details.DeliveryFormats, work.DeliveryFormats) ||
			len(detail.Attachments) != 1 || detail.Attachments[0].Name != "brief.pdf" || detail.Attachments[0].URL == "") {
			t.Fatalf("missing work detail: %s", r.Body.String())
		}
		if scenario.id == internshipID && (detail.Details.Internship == nil || detail.Details.Internship.DurationMonths != 3 || detail.PublishedAt != nil) {
			t.Fatalf("invalid internship detail: %s", r.Body.String())
		}
	}

	for _, scenario := range []struct {
		name  string
		args  url.Values
		want  []uint64
		total int64
	}{
		{"newest and excludes drafts, expired and description-only matches", nil, []uint64{internshipID, workID}, 2},
		{"Persian title normalization", url.Values{"q": {marker + " طراحی گرافیک"}}, []uint64{workID}, 1},
		{"literal wildcard characters", url.Values{"q": {marker + "%"}}, nil, 0},
		{"closest deadline", url.Values{"sort": {"closest_deadline"}}, []uint64{workID, internshipID}, 2},
		{"highest amount", url.Values{"sort": {"highest_amount"}}, []uint64{workID, internshipID}, 2},
		{"pagination", url.Values{"limit": {"1"}, "offset": {"1"}}, []uint64{workID}, 2},
		{"empty page retains total", url.Values{"offset": {"2"}}, nil, 2},
		{"unpaid internship", url.Values{"max_amount_toman": {"0"}, "opportunity_type": {"internship"}, "age_group": {"18_to_23"}, "duration": {"over_month"}}, []uint64{internshipID}, 1},
		{"combined filters and multiple selections", url.Values{
			"category": {"طراحی گرافیک", "آموزش"}, "city": {"تهران", "مشهد"}, "work_mode": {"online", "hybrid"},
			"skill_level": {"beginner"}, "opportunity_type": {"work"}, "age_group": {"under_18"}, "duration": {"under_3_days"},
			"min_amount_toman": {"2400000"},
		}, []uint64{workID}, 1},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			args := scenario.args
			if args == nil {
				args = url.Values{}
			}
			if !args.Has("q") {
				args.Set("q", marker)
			}
			r := callAPI(router, "GET", "/v1/opportunities?"+args.Encode(), "", "")
			var page entities.OpportunityPage
			if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &page) != nil {
				t.Fatalf("opportunity search: %d %s", r.Code, r.Body.String())
			}
			var ids []uint64
			for _, posting := range page.Data {
				ids = append(ids, posting.ID)
			}
			if !slices.Equal(ids, scenario.want) || page.Total != scenario.total || page.Data == nil {
				t.Fatalf("got IDs %v and total %d, want %v and %d", ids, page.Total, scenario.want, scenario.total)
			}
		})
	}
	for _, query := range []string{"sort=invalid", "duration=invalid", "limit=0", "min_amount_toman=2&max_amount_toman=1"} {
		if r := callAPI(router, "GET", "/v1/opportunities?"+query, "", ""); r.Code != 400 {
			t.Fatalf("invalid query %q returned %d", query, r.Code)
		}
	}
}

func checkJobSeekerOpportunityDashboard(t *testing.T, tx *gorm.DB, router http.Handler, tokens *security.JWTManager, marker string, workID, internshipID, draftID uint64) {
	t.Helper()
	newAccount := func() string {
		t.Helper()
		phone, national, err := fixtureIdentity()
		if err != nil {
			t.Fatal(err)
		}
		account := models.JobSeekerModel{
			Phone: phone, NationalCode: national, FirstName: "Test", LastName: "Seeker",
			Gender: "male", BirthDate: "1386/01/01", City: "تهران", Interests: []enums.Interest{enums.InterestDesign},
			PhoneVerifiedAt: time.Now(),
		}
		if err := tx.Create(&account).Error; err != nil {
			t.Fatal(err)
		}
		token, err := tokens.IssueJobSeekerAccessToken(account.ID, account.SessionVersion)
		if err != nil {
			t.Fatal(err)
		}
		return "Bearer " + token.Value
	}
	first, second := newAccount(), newAccount()
	const list = "/v1/job-seekers/opportunities"
	const skills = "/v1/job-seekers/me/skills"
	if r := callAPI(router, "GET", list, "", ""); r.Code != 401 {
		t.Fatalf("anonymous dashboard request returned %d", r.Code)
	}
	listTab := func(auth, tab string, ids ...uint64) []entities.Opportunity {
		t.Helper()
		r := callAPI(router, "GET", list+"?q="+url.QueryEscape(marker)+"&tab="+tab, "", auth)
		var page entities.OpportunityPage
		if r.Code != 200 || json.Unmarshal(r.Body.Bytes(), &page) != nil || page.Total != int64(len(ids)) || len(page.Data) != len(ids) {
			t.Fatalf("dashboard tab %s: %d %s", tab, r.Code, r.Body.String())
		}
		for i, id := range ids {
			if page.Data[i].ID != id {
				t.Fatalf("dashboard tab %s returned incorrect ordering: %s", tab, r.Body.String())
			}
		}
		return page.Data
	}
	listTab(first, "all", internshipID, workID)
	listTab(first, "work", workID)
	listTab(first, "internship", internshipID)
	listTab(first, "for_me")
	if r := callAPI(router, "PUT", skills, `{"skills":["طراحي‌پست"]}`, first); r.Code != 204 {
		t.Fatalf("replace skills: %d %s", r.Code, r.Body.String())
	}
	if r := callAPI(router, "GET", skills, "", first); r.Code != 200 || !strings.Contains(r.Body.String(), "طراحي") {
		t.Fatalf("read skills: %d %s", r.Code, r.Body.String())
	}
	matches := listTab(first, "for_me", workID, internshipID)
	if matches[0].MatchPercent == nil || *matches[0].MatchPercent != 100 || matches[1].MatchPercent == nil || *matches[1].MatchPercent != 50 {
		t.Fatalf("incorrect match percentages: %+v", matches)
	}
	listTab(second, "for_me")
	for _, query := range []string{"tab=work&tab=internship", "tab=all&opportunity_type=work", "tab=invalid"} {
		if r := callAPI(router, "GET", list+"?"+query, "", first); r.Code != 400 {
			t.Fatalf("invalid dashboard tab %q returned %d", query, r.Code)
		}
	}
	if r := callAPI(router, "PUT", fmt.Sprintf("%s/%d/saved", list, draftID), "", first); r.Code != 404 {
		t.Fatalf("save draft returned %d", r.Code)
	}
	saved := fmt.Sprintf("%s/%d/saved", list, workID)
	for _, method := range []string{"PUT", "PUT"} {
		if r := callAPI(router, method, saved, "", first); r.Code != 204 {
			t.Fatalf("save opportunity: %d %s", r.Code, r.Body.String())
		}
	}
	bookmarks := listTab(first, "saved", workID)
	if bookmarks[0].Saved == nil || !*bookmarks[0].Saved {
		t.Fatal("saved opportunity did not include saved state")
	}
	listTab(second, "saved")
	if r := callAPI(router, "DELETE", saved, "", first); r.Code != 204 {
		t.Fatalf("remove bookmark: %d %s", r.Code, r.Body.String())
	}
	listTab(first, "saved")
	if r := callAPI(router, "PUT", skills, `{"skills":[]}`, first); r.Code != 204 {
		t.Fatalf("clear skills: %d %s", r.Code, r.Body.String())
	}
	listTab(first, "for_me")
}

func checkJobPostingReview(t *testing.T, router http.Handler, storage *employerTestStorage, workID uint64, workBody, authorization string) {
	t.Helper()
	const workRoute = "/v1/employer/job-postings"
	const internshipRoute = "/v1/employer/internship-postings"
	workPath := fmt.Sprintf("%s/%d", workRoute, workID)
	upload := func(path, filename, contents string) *httptest.ResponseRecorder {
		t.Helper()
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		file, err := writer.CreateFormFile("attachments", filename)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write([]byte(contents)); err != nil {
			t.Fatal(err)
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("PUT", path+"/attachments", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Authorization", authorization)
		result := httptest.NewRecorder()
		router.ServeHTTP(result, req)
		return result
	}
	if r := upload(workPath, "brief.pdf", "%PDF-1.5\n"); r.Code != 204 {
		t.Fatalf("attachment upload: %d %s", r.Code, r.Body.String())
	}
	if r := upload(workPath, "fake.pdf", "not a PDF"); r.Code != 400 {
		t.Fatalf("invalid attachment: %d %s", r.Code, r.Body.String())
	}
	if r := callAPI(router, "PUT", workPath, workBody, authorization); r.Code != 204 {
		t.Fatalf("work update: %d %s", r.Code, r.Body.String())
	}
	if r := callAPI(router, "GET", workPath, "", authorization); r.Code != 200 ||
		r.Header().Get("Cache-Control") != "no-store" ||
		!strings.Contains(r.Body.String(), `"url":"https://private.example/signed"`) ||
		!strings.Contains(r.Body.String(), `"name":"brief.pdf"`) ||
		!strings.Contains(r.Body.String(), `"budget_toman":100000`) {
		t.Fatalf("work review: %d %s", r.Code, r.Body.String())
	}
	const internship = internshipDraftBody
	if r := callAPI(router, "POST", workRoute, internship, authorization); r.Code != 400 {
		t.Fatalf("work route accepted internship fields: %d %s", r.Code, r.Body.String())
	}
	if r := callAPI(router, "POST", internshipRoute, workBody, authorization); r.Code != 400 {
		t.Fatalf("internship route accepted work fields: %d %s", r.Code, r.Body.String())
	}
	wrongType := strings.Replace(internship, `"category":`, `"opportunity_type":"work","category":`, 1)
	if r := callAPI(router, "POST", internshipRoute, wrongType, authorization); r.Code != 400 {
		t.Fatalf("internship route accepted work type: %d %s", r.Code, r.Body.String())
	}
	r := callAPI(router, "POST", internshipRoute, internship, authorization)
	var draft general.JobPostingCreatedResponse
	if r.Code != 201 || json.Unmarshal(r.Body.Bytes(), &draft) != nil || draft.ID == 0 {
		t.Fatalf("internship create: %d %s", r.Code, r.Body.String())
	}
	internshipPath := fmt.Sprintf("%s/%d", internshipRoute, draft.ID)
	if r := callAPI(router, "GET", internshipPath, "", authorization); r.Code != 200 ||
		!strings.Contains(r.Body.String(), `"opportunity_type":"internship"`) ||
		!strings.Contains(r.Body.String(), `"description":"Write posts"`) ||
		!strings.Contains(r.Body.String(), `"monthly_stipend_toman":500000`) ||
		!strings.Contains(r.Body.String(), `"attachments":[]`) {
		t.Fatalf("internship review: %d %s", r.Code, r.Body.String())
	}
	unpaid := strings.Replace(internship, `"has_monthly_stipend":true,"monthly_stipend_toman":500000`, `"has_monthly_stipend":false`, 1)
	if r := callAPI(router, "PUT", internshipPath, unpaid, authorization); r.Code != 204 {
		t.Fatalf("unpaid internship update: %d %s", r.Code, r.Body.String())
	}
	if r := callAPI(router, "GET", internshipPath, "", authorization); r.Code != 200 ||
		!strings.Contains(r.Body.String(), `"has_monthly_stipend":false`) ||
		strings.Contains(r.Body.String(), `"monthly_stipend_toman"`) {
		t.Fatalf("unpaid internship review: %d %s", r.Code, r.Body.String())
	}
	for _, mismatch := range []struct{ path, update string }{
		{fmt.Sprintf("%s/%d", internshipRoute, workID), internship},
		{fmt.Sprintf("%s/%d", workRoute, draft.ID), workBody},
	} {
		for _, method := range []string{"GET", "PUT", "DELETE"} {
			if r := callAPI(router, method, mismatch.path, mismatch.update, authorization); r.Code != 404 {
				t.Fatalf("cross-type %s %s returned %d %s", method, mismatch.path, r.Code, r.Body.String())
			}
		}
	}
	if r := upload(fmt.Sprintf("%s/%d", workRoute, draft.ID), "brief.pdf", "%PDF-1.5\n"); r.Code != 404 {
		t.Fatalf("work attachment route accepted internship ID: %d %s", r.Code, r.Body.String())
	}
	for _, path := range []string{workPath, internshipPath} {
		if r := callAPI(router, "DELETE", path, "", authorization); r.Code != 204 {
			t.Fatalf("delete draft: %d %s", r.Code, r.Body.String())
		}
		if r := callAPI(router, "GET", path, "", authorization); r.Code != 404 {
			t.Fatalf("deleted draft: %d %s", r.Code, r.Body.String())
		}
	}
	if len(storage.objects) != 1 {
		t.Fatal("deleting the work draft should remove its attachment from storage")
	}
}
