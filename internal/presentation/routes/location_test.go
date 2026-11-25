package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/presentation/routes"
	"zedlancer/backend/wire"
)

func TestLocationEndpoints(t *testing.T) {
	router := gin.New()
	routes.SetUpRoutes(router, &wire.Application{Controllers: &wire.Controllers{GeneralControllers: &wire.GeneralControllers{}}})

	for _, tt := range []struct {
		name   string
		path   string
		status int
		key    string
		values []string
	}{
		{"provinces", "/v1/provinces", http.StatusOK, "provinces", cityOrProvinceNames(enums.Provinces())},
		{"all cities", "/v1/cities", http.StatusOK, "cities", cityOrProvinceNames(enums.Cities())},
		{"filtered cities", "/v1/cities?province=" + url.QueryEscape(string(enums.ProvinceKhorasanRazavi)), http.StatusOK, "cities", cityOrProvinceNames(enums.CitiesInProvince(enums.ProvinceKhorasanRazavi))},
		{"unknown province", "/v1/cities?province=unknown", http.StatusBadRequest, "", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if response.Code != tt.status {
				t.Fatalf("status = %d, want %d: %s", response.Code, tt.status, response.Body)
			}
			if tt.key == "" {
				return
			}
			var body map[string][]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(body[tt.key], tt.values) {
				t.Fatalf("%s = %v, want %v", tt.key, body[tt.key], tt.values)
			}
		})
	}
}

func cityOrProvinceNames[T ~string](values []T) []string {
	names := make([]string, len(values))
	for i, value := range values {
		names[i] = string(value)
	}
	return names
}
