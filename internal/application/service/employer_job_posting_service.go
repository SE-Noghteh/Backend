package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

type EmployerJobPostingService struct {
	uow        ports.UnitOfWork
	individual usecase.IndividualEmployerService
	legal      usecase.LegalEmployerService
}

func NewEmployerJobPostingService(uow ports.UnitOfWork, individual usecase.IndividualEmployerService, legal usecase.LegalEmployerService) *EmployerJobPostingService {
	return &EmployerJobPostingService{uow: uow, individual: individual, legal: legal}
}

func (s *EmployerJobPostingService) verifiedEmployerID(ctx context.Context, token string) (uint64, error) {
	employer, err := s.individual.Authenticate(ctx, token)
	if err == nil && employer != nil {
		if employer.ReviewStatus != enums.EmployerApproved {
			return 0, domainerrors.ErrEmployerNotApproved
		}
		return employer.Employer.ID, nil
	}
	if err != nil && !errors.Is(err, domainerrors.ErrUnauthenticated) {
		return 0, err
	}
	legalEmployer, err := s.legal.Authenticate(ctx, token)
	if err == nil && legalEmployer != nil {
		if legalEmployer.ReviewStatus != enums.EmployerApproved {
			return 0, domainerrors.ErrEmployerNotApproved
		}
		return legalEmployer.Employer.ID, nil
	}
	if err != nil && !errors.Is(err, domainerrors.ErrUnauthenticated) {
		return 0, err
	}
	return 0, domainerrors.ErrUnauthenticated
}

func validateJobPosting(in usecase.SaveJobPosting) error {
	if strings.TrimSpace(in.Title) == "" || utf8.RuneCountInString(in.Title) > 255 || strings.TrimSpace(in.Description) == "" || len(in.Description) > 50000 {
		return domainerrors.ErrInvalidJobPosting
	}
	invalid := func(message string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidJobPosting, message) }
	d := in.Details
	if d == nil {
		return invalid("details are required")
	}
	if d.OpportunityType != "work" && d.OpportunityType != "internship" {
		return invalid("opportunity_type must be work or internship")
	}
	if strings.TrimSpace(d.Category) == "" || utf8.RuneCountInString(d.Category) > 200 {
		return invalid("category is required, maximum 200 characters")
	}
	if len(d.Skills) > 30 {
		return invalid("maximum 30 skills")
	}
	seen := map[string]bool{}
	for _, skill := range d.Skills {
		key := strings.ToLower(strings.TrimSpace(skill))
		if key == "" || utf8.RuneCountInString(key) > 100 || seen[key] {
			return invalid("skills must be distinct non-empty names of at most 100 characters")
		}
		seen[key] = true
	}
	if d.WorkMode != "online" && d.WorkMode != "onsite" && d.WorkMode != "hybrid" {
		return invalid("work_mode must be online, onsite or hybrid")
	}
	if d.MinimumAge < 14 || d.MaximumAge > 100 || d.MaximumAge < d.MinimumAge {
		return invalid("age range must be between 14 and 100")
	}
	if d.MinimumAge < 18 && (!d.AcceptUnder18 || !d.RequiresParentApproval) {
		return invalid("accepting applicants under 18 requires parent approval")
	}
	apply, err := time.Parse("2006-01-02", d.ApplyDeadline)
	if err != nil {
		return invalid("apply_deadline must use YYYY-MM-DD")
	}
	if d.OpportunityType == "work" {
		if d.Internship != nil {
			return invalid("work ads cannot contain internship terms")
		}
		deadline, err := time.Parse("2006-01-02", d.DeliveryDeadline)
		if err != nil || !deadline.After(apply) {
			return invalid("delivery_deadline must be after apply_deadline")
		}
		if d.BudgetToman <= 0 {
			return invalid("budget_toman must be positive")
		}
	} else {
		i := d.Internship
		if i == nil {
			return invalid("internship terms are required")
		}
		if d.BudgetToman != 0 || d.DeliveryDeadline != "" {
			return invalid("internships use stipend and start date instead of work budget and delivery deadline")
		}
		if strings.TrimSpace(i.LearningOutcomes) == "" || strings.TrimSpace(i.Responsibilities) == "" {
			return invalid("learning_outcomes and responsibilities are required")
		}
		start, err := time.Parse("2006-01-02", i.StartDate)
		if err != nil || !start.After(apply) {
			return invalid("start_date must be after apply_deadline")
		}
		if i.DurationMonths < 1 || i.DurationMonths > 24 || i.HoursPerWeek < 1 || i.HoursPerWeek > 168 {
			return invalid("duration must be 1-24 months and hours_per_week 1-168")
		}
		if (i.HasMonthlyStipend && i.MonthlyStipendToman <= 0) || (!i.HasMonthlyStipend && i.MonthlyStipendToman != 0) {
			return invalid("monthly stipend must match has_monthly_stipend")
		}
	}
	return nil
}

func (s *EmployerJobPostingService) Create(ctx context.Context, token string, in usecase.SaveJobPosting) (uint64, error) {
	if err := validateJobPosting(in); err != nil {
		return 0, err
	}
	employerID, err := s.verifiedEmployerID(ctx, token)
	if err != nil {
		return 0, err
	}
	details, err := json.Marshal(in.Details)
	if err != nil {
		return 0, err
	}
	posting := &entities.JobPosting{EmployerID: employerID, Title: strings.TrimSpace(in.Title), Description: strings.TrimSpace(in.Description), Details: details}
	if err := s.uow.Factory().JobPostingRepository().Create(ctx, posting); err != nil {
		return 0, err
	}
	return posting.ID, nil
}

func (s *EmployerJobPostingService) Update(ctx context.Context, token string, id uint64, in usecase.SaveJobPosting) error {
	if id == 0 {
		return domainerrors.ErrInvalidJobPosting
	}
	if err := validateJobPosting(in); err != nil {
		return err
	}
	employerID, err := s.verifiedEmployerID(ctx, token)
	if err != nil {
		return err
	}
	details, err := json.Marshal(in.Details)
	if err != nil {
		return err
	}
	updated, err := s.uow.Factory().JobPostingRepository().UpdateOwned(ctx, id, employerID, strings.TrimSpace(in.Title), strings.TrimSpace(in.Description), details)
	if err != nil {
		return err
	}
	if !updated {
		return domainerrors.ErrJobPostingNotFound
	}
	return nil
}

func (s *EmployerJobPostingService) Delete(ctx context.Context, token string, id uint64) error {
	if id == 0 {
		return domainerrors.ErrInvalidJobPosting
	}
	employerID, err := s.verifiedEmployerID(ctx, token)
	if err != nil {
		return err
	}
	deleted, err := s.uow.Factory().JobPostingRepository().DeleteOwned(ctx, id, employerID)
	if err != nil {
		return err
	}
	if !deleted {
		return domainerrors.ErrJobPostingNotFound
	}
	return nil
}
