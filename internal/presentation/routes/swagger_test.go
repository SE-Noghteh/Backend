package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

func TestSwaggerServesUIAndSignupContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	app := &wire.Application{Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{JobSeekerController: general.NewJobSeekerController(nil), JobSeekerAuthController: general.NewJobSeekerAuthController(nil)}}}
	routes.SetUpRoutes(router, app)
	for _, path := range []string{"/swagger/index.html", "/swagger/swagger-initializer.js", "/swagger/swagger-ui-bundle.js", "/swagger/swagger-ui.css", "/swagger/doc.json", "/v1/job-seekers/signup-options", "/v1/legal-employers/signup-options"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s returned %d", path, response.Code)
		}
		if path == "/swagger/swagger-initializer.js" && !strings.Contains(response.Body.String(), "/swagger/doc.json") {
			t.Fatal("Swagger UI must load the local specification")
		}
		if path != "/swagger/doc.json" {
			continue
		}
		var spec struct {
			BasePath    string                                `json:"basePath"`
			Host        string                                `json:"host"`
			Paths       map[string]map[string]json.RawMessage `json:"paths"`
			Definitions map[string]struct {
				Enum     []string `json:"enum"`
				Required []string `json:"required"`
			} `json:"definitions"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &spec); err != nil {
			t.Fatal(err)
		}
		if spec.BasePath != "/v1" || spec.Host != "" {
			t.Fatal("Swagger must use /v1 on the current host and port")
		}
		for path, method := range map[string]string{"/job-seekers/signup-options": "get", "/job-seekers/otp": "post", "/job-seekers/signup": "post", "/job-seekers/verify-otp": "post", "/job-seekers/login": "post", "/job-seekers/me": "get", "/job-seekers/logout": "post"} {
			if spec.Paths[path][method] == nil {
				t.Fatalf("missing documented endpoint %s %s", method, path)
			}
		}
		if len(spec.Definitions["enums.City"].Enum) != 10 || len(spec.Definitions["enums.Interest"].Enum) != 10 {
			t.Fatal("all signup choices must be documented")
		}
		var seekerSignup struct {
			Consumes   []string `json:"consumes"`
			Parameters []struct {
				Name             string `json:"name"`
				In               string `json:"in"`
				Type             string `json:"type"`
				Required         bool   `json:"required"`
				CollectionFormat string `json:"collectionFormat"`
			} `json:"parameters"`
		}
		if err := json.Unmarshal(spec.Paths["/job-seekers/signup"]["post"], &seekerSignup); err != nil {
			t.Fatal(err)
		}
		if len(seekerSignup.Consumes) != 1 || seekerSignup.Consumes[0] != "multipart/form-data" || len(seekerSignup.Parameters) != 13 {
			t.Fatal("job seeker signup must accept the profile and required video in one multipart submission")
		}
		for _, p := range seekerSignup.Parameters {
			if p.In != "formData" || !p.Required {
				t.Fatalf("missing required form field %s", p.Name)
			}
			if p.Name == "verification_video" && p.Type != "file" {
				t.Fatal("video must be a file")
			}
			if p.Name == "interests" && p.CollectionFormat != "multi" {
				t.Fatal("interests must support repeated fields")
			}
		}
		if spec.Paths["/job-seekers/documents/video"]["get"] == nil {
			t.Fatal("missing private job seeker video route")
		}
		for path, method := range map[string]string{"/job-seekers/phone-change": "post", "/job-seekers/phone-change/verify": "post", "/job-seekers/me": "delete"} {
			var operation struct {
				Security []map[string][]string `json:"security"`
			}
			if err := json.Unmarshal(spec.Paths[path][method], &operation); err != nil {
				t.Fatal(err)
			}
			if len(operation.Security) != 1 || operation.Security[0]["BearerAuth"] == nil {
				t.Fatalf("missing bearer security for %s %s", method, path)
			}
		}
		for _, path := range []string{"/job-seekers/parent-approval", "/parents/invitations/validate", "/parents/signup", "/parents/otp", "/parents/verify-otp"} {
			if spec.Paths[path]["post"] == nil {
				t.Fatalf("missing parent endpoint %s", path)
			}
		}
		var parentSignup struct {
			Parameters []struct {
				Name     string `json:"name"`
				Required bool   `json:"required"`
			} `json:"parameters"`
		}
		if err := json.Unmarshal(spec.Paths["/parents/signup"]["post"], &parentSignup); err != nil {
			t.Fatal(err)
		}
		if len(parentSignup.Parameters) != 6 {
			t.Fatal("parent signup must have exactly six fields")
		}
		for _, p := range parentSignup.Parameters {
			if !p.Required || p.Name == "first_name" || p.Name == "last_name" || p.Name == "phone" {
				t.Fatal("parent signup must use the saved identity and require all form fields")
			}
		}
		var employerSignup struct {
			Consumes   []string `json:"consumes"`
			Parameters []struct {
				Name     string `json:"name"`
				In       string `json:"in"`
				Type     string `json:"type"`
				Required bool   `json:"required"`
			} `json:"parameters"`
		}
		if err := json.Unmarshal(spec.Paths["/individual-employers/signup"]["post"], &employerSignup); err != nil {
			t.Fatal(err)
		}
		if len(employerSignup.Consumes) != 1 || employerSignup.Consumes[0] != "multipart/form-data" {
			t.Fatal("employer signup must be one multipart request")
		}
		files := map[string]bool{}
		for _, param := range employerSignup.Parameters {
			if param.Name == "otp" || param.Name == "confirm_password" || param.Name == "accept_terms" {
				t.Fatal("signup must not require OTP or frontend-only fields")
			}
			if param.Type == "file" {
				files[param.Name] = param.Required
			}
		}
		if len(files) != 3 || !files["identity_document"] || !files["verification_video"] || files["business_license"] {
			t.Fatal("employer document requirements are incorrect")
		}
		for path, method := range map[string]string{"/individual-employers/signup-options": "get", "/individual-employers/otp": "post", "/individual-employers/verify-otp": "post", "/individual-employers/login": "post", "/individual-employers/logout": "post", "/individual-employers/me": "get", "/individual-employers/documents/{kind}": "get"} {
			if spec.Paths[path][method] == nil {
				t.Fatalf("missing employer route %s", path)
			}
		}

		var legalSignup struct {
			Consumes   []string `json:"consumes"`
			Parameters []struct {
				Name             string `json:"name"`
				Type             string `json:"type"`
				Required         bool   `json:"required"`
				CollectionFormat string `json:"collectionFormat"`
			} `json:"parameters"`
		}
		if err := json.Unmarshal(spec.Paths["/legal-employers/signup"]["post"], &legalSignup); err != nil {
			t.Fatal(err)
		}
		if len(legalSignup.Consumes) != 1 || legalSignup.Consumes[0] != "multipart/form-data" {
			t.Fatal("legal signup must accept multipart")
		}
		legalFiles := map[string]bool{}
		multiActivities := false
		for _, param := range legalSignup.Parameters {
			if param.Name == "otp" || param.Name == "confirm_password" || param.Name == "accept_terms" {
				t.Fatal("legal signup exposes frontend-only fields or OTP")
			}
			if param.Type == "file" {
				legalFiles[param.Name] = param.Required
			}
			if param.Name == "activities" {
				multiActivities = param.Type == "array" && param.Required && param.CollectionFormat == "multi"
			}
		}
		if len(legalFiles) != 5 || !legalFiles["establishment_document"] || !legalFiles["identity_document"] || !legalFiles["verification_video"] || legalFiles["activity_license"] || legalFiles["authorization_document"] {
			t.Fatal("incorrect legal document contract")
		}
		if !multiActivities {
			t.Fatal("legal activities must accept multiple form values")
		}
		for path, method := range map[string]string{"/legal-employers/signup-options": "get", "/legal-employers/otp": "post", "/legal-employers/verify-otp": "post", "/legal-employers/login": "post", "/legal-employers/logout": "post", "/legal-employers/me": "get", "/legal-employers/documents/{kind}": "get"} {
			if spec.Paths[path][method] == nil {
				t.Fatalf("missing legal route %s", path)
			}
		}

	}
}
