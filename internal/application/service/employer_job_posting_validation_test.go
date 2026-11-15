package service

import (
	"errors"
	"testing"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

func TestJobPostingTermsValidation(t *testing.T) {
	for _, scenario := range []string{"work", "internship", "missing terms", "minor without consent", "reversed deadlines", "unpaid with stipend"} {
		t.Run(scenario, func(t *testing.T) {
			d := &entities.JobPostingDetails{OpportunityType: "work", Category: "design", WorkMode: "online", ApplyDeadline: "2027-01-01", DeliveryDeadline: "2027-02-01", BudgetToman: 2400000, MinimumAge: 14, MaximumAge: 18, AcceptUnder18: true, RequiresParentApproval: true}
			in := usecase.SaveJobPosting{Title: "طراحی پست", Description: "Design ten posts", Details: d}
			switch scenario {
			case "missing terms":
				in.Details = nil
			case "minor without consent":
				d.RequiresParentApproval = false
			case "reversed deadlines":
				d.DeliveryDeadline = "2026-12-01"
			case "internship", "unpaid with stipend":
				d.OpportunityType, d.DeliveryDeadline, d.BudgetToman = "internship", "", 0
				d.Internship = &entities.InternshipTerms{LearningOutcomes: "Content planning", Responsibilities: "Write posts", StartDate: "2027-02-01", DurationMonths: 3, HoursPerWeek: 6}
				if scenario == "unpaid with stipend" {
					d.Internship.MonthlyStipendToman = 100
				}
			}
			err := validateJobPosting(in)
			valid := scenario == "work" || scenario == "internship"
			if valid && err != nil {
				t.Fatal(err)
			}
			if !valid && !errors.Is(err, domainerrors.ErrInvalidJobPosting) {
				t.Fatalf("expected invalid posting, got %v", err)
			}
		})
	}
}
