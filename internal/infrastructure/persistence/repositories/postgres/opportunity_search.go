package postgres

import (
	"context"
	"errors"
	"slices"
	"time"

	"gorm.io/gorm"

	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

// expression is a SQL column name from this repository, never a request value.
func normalizedOpportunityColumn(expression string) string {
	return "btrim(regexp_replace(translate(lower(" + expression + "), 'يك‌', 'یک '), '[[:space:]]+', ' ', 'g'))"
}

const openOpportunityDeadline = "details->>'apply_deadline' >= to_char(CURRENT_TIMESTAMP AT TIME ZONE 'Asia/Tehran', 'YYYY-MM-DD')"

const opportunitySummaryColumns = `
			job_postings.id, job_postings.title, job_postings.created_at,
			COALESCE(legal.company_name, individual.business_name, '') AS employer_name,
			COALESCE(legal.city, individual.city, '') AS city,
			details->>'category' AS category,
			details->>'work_mode' AS work_mode,
			details->>'opportunity_type' AS opportunity_type,
			details->>'skill_level' AS skill_level,
			(details->>'minimum_age')::integer AS minimum_age,
			(details->>'maximum_age')::integer AS maximum_age,
			COALESCE((details->>'accept_under_18')::boolean, false) AS accept_under_18,
			COALESCE((details->>'requires_parent_approval')::boolean, false) AS requires_parent_approval,
			details->>'apply_deadline' AS apply_deadline,
			CASE details->>'opportunity_type'
				WHEN 'work' THEN COALESCE((details->>'budget_toman')::bigint, 0)
				WHEN 'internship' THEN COALESCE((details#>>'{internship,monthly_stipend_toman}')::bigint, 0)
			END AS amount_toman,
			CASE details->>'opportunity_type'
				WHEN 'work' THEN COALESCE(NULLIF((details->>'duration_days')::bigint, 0),
					(details->>'delivery_deadline')::date - (details->>'apply_deadline')::date)
				WHEN 'internship' THEN (details#>>'{internship,duration_months}')::integer * 30
			END AS duration_days`

func publishedOpportunities(db *gorm.DB) *gorm.DB {
	return db.Model(&models.JobPostingModel{}).
		Joins("LEFT JOIN individual_employers AS individual ON individual.employer_id = job_postings.employer_id").
		Joins("LEFT JOIN legal_employers AS legal ON legal.employer_id = job_postings.employer_id").
		Where("job_postings.status = ? AND details->>'opportunity_type' IN ?", "published", []entities.JobPostingType{entities.WorkPosting, entities.InternshipPosting})
}

func (r *JobPostingRepository) ListPublished(ctx context.Context, filters entities.OpportunityFilters) (*entities.OpportunityPage, error) {
	return r.listPublished(ctx, filters, nil)
}

func (r *JobPostingRepository) ListPublishedForJobSeeker(ctx context.Context, filters entities.OpportunityFilters, scope entities.OpportunityDashboard) (*entities.OpportunityPage, error) {
	return r.listPublished(ctx, filters, &scope)
}

func (r *JobPostingRepository) listPublished(ctx context.Context, filters entities.OpportunityFilters, scope *entities.OpportunityDashboard) (*entities.OpportunityPage, error) {
	db := r.db.WithContext(ctx)
	published := publishedOpportunities(db).Where(openOpportunityDeadline)
	if scope == nil {
		published = published.Select(opportunitySummaryColumns)
	} else {
		published = selectDashboardOpportunities(published, *scope)
	}
	query := filterOpportunities(db.Table("(?) AS opportunities", published), filters)
	if scope != nil {
		switch scope.Tab {
		case entities.OpportunityTabForMe:
			query = query.Where("match_percent > 0")
		case entities.OpportunityTabSaved:
			query = query.Where("is_saved")
		}
	}
	page := &entities.OpportunityPage{Data: []entities.Opportunity{}, Limit: filters.Limit, Offset: filters.Offset}
	if err := query.Session(&gorm.Session{}).Count(&page.Total).Error; err != nil {
		return nil, err
	}
	switch filters.Sort {
	case entities.OpportunityClosestDeadline:
		query = query.Order("apply_deadline ASC")
	case entities.OpportunityHighestAmount:
		query = query.Order("amount_toman DESC")
	case entities.OpportunityBestMatch:
		query = query.Order("match_percent DESC")
	}
	if err := query.Order("created_at DESC, id DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&page.Data).Error; err != nil {
		return nil, err
	}
	return page, nil
}

func selectDashboardOpportunities(query *gorm.DB, scope entities.OpportunityDashboard) *gorm.DB {
	columns := opportunitySummaryColumns + `, EXISTS (
		SELECT 1 FROM saved_job_postings
		WHERE job_seeker_id = ? AND job_posting_id = job_postings.id
	) AS is_saved`
	if len(scope.Skills) == 0 {
		return query.Select(columns+", 0 AS match_percent", scope.JobSeekerID)
	}
	normalized := make([]string, 0, len(scope.Skills))
	for _, skill := range scope.Skills {
		normalized = append(normalized, entities.NormalizeOpportunityText(skill))
	}
	skillName := normalizedOpportunityColumn("posting_skill.name")
	return query.Select(columns+`, (
		SELECT COALESCE(100 * COUNT(DISTINCT `+skillName+`) FILTER (WHERE `+skillName+` IN ?) /
			NULLIF(COUNT(DISTINCT `+skillName+`), 0), 0)
		FROM jsonb_array_elements_text(CASE WHEN jsonb_typeof(details->'skills') = 'array'
			THEN details->'skills' ELSE '[]'::jsonb END) AS posting_skill(name)
		WHERE `+skillName+` <> ''
	) AS match_percent`, scope.JobSeekerID, normalized)
}

func (r *JobPostingRepository) SaveForJobSeeker(ctx context.Context, seekerID, postingID uint64) error {
	result := r.db.WithContext(ctx).Exec(`
		INSERT INTO saved_job_postings (job_seeker_id, job_posting_id)
		SELECT ?, id FROM job_postings
		WHERE id = ? AND status = 'published' AND details->>'opportunity_type' IN ('work', 'internship')
		ON CONFLICT (job_seeker_id, job_posting_id) DO UPDATE SET job_posting_id = EXCLUDED.job_posting_id`, seekerID, postingID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return domainerrors.ErrJobPostingNotFound
	}
	return nil
}

func (r *JobPostingRepository) UnsaveForJobSeeker(ctx context.Context, seekerID, postingID uint64) error {
	return r.db.WithContext(ctx).Where("job_seeker_id = ? AND job_posting_id = ?", seekerID, postingID).
		Delete(&models.SavedJobPostingModel{}).Error
}

func (r *JobPostingRepository) FindPublishedOpportunity(ctx context.Context, id uint64) (*entities.PublishedOpportunity, error) {
	var row struct {
		entities.Opportunity
		Description           string
		Details               *entities.JobPostingDetails `gorm:"serializer:json"`
		EmployerVerified      bool
		AcceptingApplications bool
		PublishedAt           *time.Time
		Attachments           []entities.JobPostingAttachment `gorm:"serializer:json"`
	}
	err := publishedOpportunities(r.db.WithContext(ctx)).
		Select(opportunitySummaryColumns+`,
			job_postings.description, job_postings.details, job_postings.published_at, job_postings.attachments,
			COALESCE(legal.review_status, individual.review_status, '') = 'approved' AS employer_verified,
			COALESCE(`+openOpportunityDeadline+`, false) AS accepting_applications`).
		Where("job_postings.id = ?", id).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domainerrors.ErrJobPostingNotFound
	}
	if err != nil {
		return nil, err
	}
	return &entities.PublishedOpportunity{
		Opportunity: row.Opportunity, Description: row.Description, Details: row.Details,
		EmployerVerified: row.EmployerVerified, AcceptingApplications: row.AcceptingApplications,
		PublishedAt: row.PublishedAt, Attachments: row.Attachments,
	}, nil
}

func filterOpportunities(query *gorm.DB, f entities.OpportunityFilters) *gorm.DB {
	if title := entities.NormalizeOpportunityText(f.Title); title != "" {
		query = query.Where("strpos("+normalizedOpportunityColumn("title")+", ?) > 0", title)
	}
	for _, filter := range []struct {
		column string
		values []string
	}{{"category", f.Categories}, {"city", f.Cities}} {
		if len(filter.values) > 0 {
			values := make([]string, len(filter.values))
			for i, value := range filter.values {
				values[i] = entities.NormalizeOpportunityText(value)
			}
			query = query.Where(normalizedOpportunityColumn(filter.column)+" IN ?", values)
		}
	}
	if len(f.Types) > 0 {
		query = query.Where("opportunity_type IN ?", f.Types)
	}
	if len(f.WorkModes) > 0 {
		query = query.Where("work_mode IN ?", f.WorkModes)
	}
	if len(f.SkillLevels) > 0 {
		query = query.Where("skill_level IN ?", f.SkillLevels)
	}
	if len(f.AgeGroups) > 0 {
		minors := slices.Contains(f.AgeGroups, entities.OpportunityUnder18)
		adults := slices.Contains(f.AgeGroups, entities.Opportunity18To23)
		query = query.Where(`((? AND minimum_age <= 17 AND maximum_age >= 14 AND accept_under_18 AND requires_parent_approval)
			OR (? AND minimum_age <= 23 AND maximum_age >= 18))`, minors, adults)
	}
	if f.MinAmountToman != nil {
		query = query.Where("amount_toman >= ?", *f.MinAmountToman)
	}
	if f.MaxAmountToman != nil {
		query = query.Where("amount_toman <= ?", *f.MaxAmountToman)
	}
	if len(f.Durations) > 0 {
		query = query.Where(`CASE
			WHEN duration_days < 3 THEN 'under_3_days'
			WHEN duration_days <= 7 THEN 'up_to_week'
			WHEN duration_days <= 30 THEN 'up_to_month'
			WHEN duration_days > 30 THEN 'over_month'
			END IN ?`, f.Durations)
	}
	return query
}
