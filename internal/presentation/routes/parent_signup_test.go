package routes_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

type parentAPI struct {
	id           uint64
	code, otp    string
	stage        usecase.JobSeekerParentInput
	signup       usecase.SignupParent
	image, video []byte
	calls        int
	err          error
}

func (s *parentAPI) SetJobSeekerParent(_ context.Context, id uint64, in usecase.JobSeekerParentInput) (*usecase.ParentStageResult, error) {
	s.calls++
	s.id, s.stage = id, in
	return &usecase.ParentStageResult{SignupStatus: entities.SignupFirstStageCompleted, InvitationCode: "invitation"}, s.err
}
func (s *parentAPI) ValidateInvitation(_ context.Context, code string) (*usecase.ParentInvitationInfo, error) {
	s.calls++
	s.code = code
	return &usecase.ParentInvitationInfo{FirstName: "Saved", LastName: "Parent", Phone: "09129876543"}, s.err
}
func (s *parentAPI) Signup(_ context.Context, in usecase.SignupParent) error {
	s.calls++
	s.signup = in
	s.image, _ = io.ReadAll(in.IdentityDocument.Body)
	s.video, _ = io.ReadAll(in.ConsentVideo.Body)
	return s.err
}
func (s *parentAPI) ResendOTP(_ context.Context, code string) error {
	s.calls++
	s.code = code
	return s.err
}
func (s *parentAPI) VerifyOTP(_ context.Context, code, otp string) (*entities.Parent, error) {
	s.calls++
	s.code, s.otp = code, otp
	return &entities.Parent{ID: 9, FirstName: "Saved", LastName: "Parent", Phone: "09129876543", Relationship: "father", PasswordHash: "must-not-leak"}, s.err
}
func parentRouter(s *parentAPI) *gin.Engine {
	gin.SetMode(gin.TestMode)
	auth := &accountAPI{}
	app := &wire.Application{JobSeekerAuthService: auth, Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{ParentController: general.NewParentController(s), JobSeekerController: general.NewJobSeekerController(nil), JobSeekerAuthController: general.NewJobSeekerAuthController(auth)}}}
	router := gin.New()
	routes.SetUpRoutes(router, app)
	return router
}
func TestParentJSONRoutes(t *testing.T) {
	for _, tc := range []struct {
		path, body string
		status     int
		auth       bool
	}{
		{"/v1/job-seekers/parent-approval", `{"first_name":"Saved","last_name":"Parent","phone":"09129876543"}`, 200, true},
		{"/v1/parents/invitations/validate", `{"code":"invitation"}`, 200, false},
		{"/v1/parents/otp", `{"code":"invitation"}`, 202, false},
		{"/v1/parents/verify-otp", `{"code":"invitation","otp":"123456"}`, 201, false},
	} {
		for _, scenario := range []string{"success", "malformed", "service error"} {
			t.Run(tc.path+scenario, func(t *testing.T) {
				s := &parentAPI{}
				router := parentRouter(s)
				body, want := tc.body, tc.status
				if scenario == "malformed" {
					body, want = "{", 400
				}
				if scenario == "service error" {
					s.err, want = domainerrors.ErrInvalidInvitation, 400
				}
				req := httptest.NewRequest(http.MethodPost, tc.path, strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				if tc.auth {
					req.Header.Set("Authorization", "Bearer test-token")
				}
				r := httptest.NewRecorder()
				router.ServeHTTP(r, req)
				if r.Code != want {
					t.Fatalf("%d: %s", r.Code, r.Body.String())
				}
				if scenario == "malformed" {
					if s.calls != 0 {
						t.Fatal("bad input reached service")
					}
					return
				}
				if s.calls != 1 || r.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("incorrect service call or response caching")
				}
				if tc.auth && (s.id != 7 || s.stage.Phone != "09129876543") {
					t.Fatal("stage two did not use authenticated child")
				}
				if !tc.auth && s.code != "invitation" {
					t.Fatal("invitation code missing")
				}
				if strings.HasSuffix(tc.path, "verify-otp") && (s.otp != "123456" || strings.Contains(r.Body.String(), "must-not-leak")) {
					t.Fatal("OTP not passed or sensitive fields leaked")
				}
			})
		}
	}
	s := &parentAPI{}
	router := parentRouter(s)
	r := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/job-seekers/parent-approval", strings.NewReader(`{"skip":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(r, req)
	if r.Code != 401 || s.calls != 0 {
		t.Fatal("anonymous stage-two request accepted")
	}
}
func TestParentSignupMultipart(t *testing.T) {
	for _, scenario := range []string{"success", "missing image", "duplicate video", "missing code", "JSON"} {
		t.Run(scenario, func(t *testing.T) {
			s := &parentAPI{}
			router := parentRouter(s)
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			fields := map[string]string{"code": "invitation", "relationship": "mother", "password": "parent-password", "consent_accepted": "true", "first_name": "Ignored", "phone": "09120000000"}
			if scenario == "missing code" {
				delete(fields, "code")
			}
			for k, v := range fields {
				if err := writer.WriteField(k, v); err != nil {
					t.Fatal(err)
				}
			}
			files := []struct{ field, name, body string }{{"identity_document", "identity.png", "image"}, {"consent_video", "consent.mp4", "video"}}
			if scenario == "missing image" {
				files = files[1:]
			}
			if scenario == "duplicate video" {
				files = append(files, files[1])
			}
			for _, file := range files {
				part, err := writer.CreateFormFile(file.field, file.name)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := io.WriteString(part, file.body); err != nil {
					t.Fatal(err)
				}
			}
			if err := writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/v1/parents/signup", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if scenario == "JSON" {
				req = httptest.NewRequest(http.MethodPost, "/v1/parents/signup", strings.NewReader(`{"code":"invitation"}`))
				req.Header.Set("Content-Type", "application/json")
			}
			r := httptest.NewRecorder()
			router.ServeHTTP(r, req)
			if scenario != "success" {
				if r.Code != 400 || s.calls != 0 {
					t.Fatalf("invalid signup accepted: %d", r.Code)
				}
				return
			}
			if r.Code != 202 || s.calls != 1 || s.signup.Code != "invitation" || s.signup.Relationship != "mother" || !s.signup.ConsentAccepted || s.signup.Password != "parent-password" || string(s.image) != "image" || string(s.video) != "video" {
				t.Fatalf("multipart fields/files not bound: %d %s", r.Code, r.Body.String())
			}
		})
	}
}
