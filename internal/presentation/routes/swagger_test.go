package routes_test

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"zedlancer/backend/internal/presentation/controllers/v1/general"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

func TestSwaggerServesUIAndSignupContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	app := &wire.Application{Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{JobSeekerController: general.NewJobSeekerController(nil), JobSeekerAuthController: general.NewJobSeekerAuthController(nil)}}}
	routes.SetUpRoutes(router, app)
	for _, path := range []string{"/swagger/index.html", "/swagger/swagger-initializer.js", "/swagger/swagger-ui-bundle.js", "/swagger/swagger-ui.css", "/swagger/doc.json", "/v1/job-seekers/signup-options"} {
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
		if len(spec.Definitions["entities.City"].Enum) != 10 || len(spec.Definitions["entities.Interest"].Enum) != 10 {
			t.Fatal("all signup choices must be documented")
		}
		if len(spec.Definitions["usecase.SignupJobSeeker"].Required) != 10 {
			t.Fatal("all signup fields must be marked required")
		}
	}
}
