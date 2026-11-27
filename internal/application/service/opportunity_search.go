package service

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

func (s *EmployerJobPostingService) GetOpportunity(ctx context.Context, id uint64) (*usecase.OpportunityDetail, error) {
	posting, err := s.uow.Factory().JobPostingRepository().FindPublishedOpportunity(ctx, id)
	if err != nil {
		return nil, err
	}
	attachments, err := s.attachmentDownloads(ctx, posting.Attachments)
	if err != nil {
		return nil, err
	}
	return &usecase.OpportunityDetail{PublishedOpportunity: posting, Attachments: attachments}, nil
}

func (s *EmployerJobPostingService) ListOpportunities(ctx context.Context, filters entities.OpportunityFilters) (*entities.OpportunityPage, error) {
	filters = defaultOpportunityFilters(filters)
	if err := validateOpportunityFilters(filters, false); err != nil {
		return nil, err
	}
	return s.uow.Factory().JobPostingRepository().ListPublished(ctx, filters)
}

func (s *EmployerJobPostingService) ListJobSeekerOpportunities(ctx context.Context, seeker *entities.JobSeeker, tab entities.OpportunityTab, filters entities.OpportunityFilters) (*entities.OpportunityPage, error) {
	switch tab {
	case "", entities.OpportunityTabAll:
		tab = entities.OpportunityTabAll
	case entities.OpportunityTabForMe:
		if filters.Sort == "" {
			filters.Sort = entities.OpportunityBestMatch
		}
	case entities.OpportunityTabWork:
		filters.Types = []entities.JobPostingType{entities.WorkPosting}
	case entities.OpportunityTabInternship:
		filters.Types = []entities.JobPostingType{entities.InternshipPosting}
	case entities.OpportunityTabSaved:
	default:
		return nil, fmt.Errorf("%w: tab", domainerrors.ErrInvalidOpportunityFilters)
	}
	filters = defaultOpportunityFilters(filters)
	if err := validateOpportunityFilters(filters, true); err != nil {
		return nil, err
	}
	return s.uow.Factory().JobPostingRepository().ListPublishedForJobSeeker(ctx, filters, entities.OpportunityDashboard{
		JobSeekerID: seeker.ID, Tab: tab, Skills: seeker.Skills,
	})
}

func (s *EmployerJobPostingService) UpdateJobSeekerSkills(ctx context.Context, id uint64, skills []string) error {
	if len(skills) > 30 {
		return fmt.Errorf("%w: maximum 30 skills", domainerrors.ErrInvalidOpportunityFilters)
	}
	seen := make(map[string]bool, len(skills))
	names := make([]string, 0, len(skills))
	for _, skill := range skills {
		name := strings.TrimSpace(skill)
		key := entities.NormalizeOpportunityText(name)
		if key == "" || utf8.RuneCountInString(name) > 100 || seen[key] {
			return fmt.Errorf("%w: skills must be distinct nonempty names of at most 100 characters", domainerrors.ErrInvalidOpportunityFilters)
		}
		seen[key] = true
		names = append(names, name)
	}
	return s.uow.Factory().JobSeekerRepository().UpdateSkills(ctx, id, names)
}

func (s *EmployerJobPostingService) SaveOpportunity(ctx context.Context, seekerID, postingID uint64) error {
	return s.uow.Factory().JobPostingRepository().SaveForJobSeeker(ctx, seekerID, postingID)
}

func (s *EmployerJobPostingService) UnsaveOpportunity(ctx context.Context, seekerID, postingID uint64) error {
	return s.uow.Factory().JobPostingRepository().UnsaveForJobSeeker(ctx, seekerID, postingID)
}

func defaultOpportunityFilters(filters entities.OpportunityFilters) entities.OpportunityFilters {
	filters.Title = strings.TrimSpace(filters.Title)
	if filters.Sort == "" {
		filters.Sort = entities.OpportunityNewest
	}
	if filters.Limit == 0 {
		filters.Limit = 20
	}
	return filters
}

func validateOpportunityFilters(f entities.OpportunityFilters, dashboard bool) error {
	invalid := func(field string) error {
		return fmt.Errorf("%w: %s", domainerrors.ErrInvalidOpportunityFilters, field)
	}
	if utf8.RuneCountInString(f.Title) > 255 {
		return invalid("q exceeds 255 characters")
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 {
		return invalid("limit must be 1-100 and offset must be nonnegative")
	}
	switch f.Sort {
	case entities.OpportunityNewest, entities.OpportunityClosestDeadline, entities.OpportunityHighestAmount:
	case entities.OpportunityBestMatch:
		if !dashboard {
			return invalid("sort")
		}
	default:
		return invalid("sort")
	}
	for _, category := range f.Categories {
		if strings.TrimSpace(category) == "" || utf8.RuneCountInString(category) > 200 {
			return invalid("category")
		}
	}
	for _, city := range f.Cities {
		if strings.TrimSpace(city) == "" || utf8.RuneCountInString(city) > 40 {
			return invalid("city")
		}
	}
	for _, mode := range f.WorkModes {
		if mode != "online" && mode != "onsite" && mode != "hybrid" {
			return invalid("work_mode")
		}
	}
	for _, kind := range f.Types {
		if kind != entities.WorkPosting && kind != entities.InternshipPosting {
			return invalid("opportunity_type")
		}
	}
	for _, group := range f.AgeGroups {
		if group != entities.OpportunityUnder18 && group != entities.Opportunity18To23 {
			return invalid("age_group")
		}
	}
	for _, level := range f.SkillLevels {
		if !level.Valid() {
			return invalid("skill_level")
		}
	}
	for _, duration := range f.Durations {
		if !slices.Contains([]entities.OpportunityDuration{
			entities.OpportunityUnderThreeDays, entities.OpportunityUpToWeek,
			entities.OpportunityUpToMonth, entities.OpportunityOverMonth,
		}, duration) {
			return invalid("duration")
		}
	}
	if (f.MinAmountToman != nil && *f.MinAmountToman < 0) ||
		(f.MaxAmountToman != nil && *f.MaxAmountToman < 0) ||
		(f.MinAmountToman != nil && f.MaxAmountToman != nil && *f.MinAmountToman > *f.MaxAmountToman) {
		return invalid("amount must be nonnegative and minimum cannot exceed maximum")
	}
	return nil
}
