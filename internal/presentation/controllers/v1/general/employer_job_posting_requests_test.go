package general

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/domain/entities"
)

func TestBindJobPostingUsesRouteType(t *testing.T) {
	for _, test := range []struct {
		name    string
		kind    entities.JobPostingType
		details string
		wantErr bool
	}{
		{"work", entities.WorkPosting, `"budget_toman":100000,"delivery_deadline":"2028-02-15"`, false},
		{"internship", entities.InternshipPosting, `"internship":{"learning_outcomes":"Plan content","has_monthly_stipend":false}`, false},
		{"internship terms on work route", entities.WorkPosting, `"internship":null`, true},
		{"work budget on internship route", entities.InternshipPosting, `"budget_toman":0`, true},
		{"wrong explicit type", entities.InternshipPosting, `"opportunity_type":"work"`, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			body := `{"title":"Test","details":{"category":"Content",` + test.details + `}}`
			ctx.Request = httptest.NewRequest("POST", "/", strings.NewReader(body))
			got, err := bindJobPosting(ctx, test.kind)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected the route to reject the request")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.Details.OpportunityType != test.kind || got.Details.Category != "Content" {
				t.Fatalf("wrong details: %+v", got.Details)
			}
		})
	}
}
