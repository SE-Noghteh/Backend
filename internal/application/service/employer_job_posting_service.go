package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type EmployerJobPostingService struct {
	uow        ports.UnitOfWork
	individual usecase.IndividualEmployerService
	legal      usecase.LegalEmployerService
	storage    ports.Storage
	media      ports.JobPostingAttachmentValidator
}

func NewEmployerJobPostingService(uow ports.UnitOfWork, individual usecase.IndividualEmployerService, legal usecase.LegalEmployerService, storage ports.Storage, media ports.JobPostingAttachmentValidator) *EmployerJobPostingService {
	return &EmployerJobPostingService{uow: uow, individual: individual, legal: legal, storage: storage, media: media}
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
	invalid := func(message string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidJobPosting, message) }
	if strings.TrimSpace(in.Title) == "" || utf8.RuneCountInString(in.Title) > 255 {
		return invalid("title is required, maximum 255 characters")
	}
	if len(in.Description) > 50000 {
		return invalid("description cannot exceed 50000 bytes")
	}
	d := in.Details
	if d == nil {
		return invalid("details are required")
	}
	if d.OpportunityType != entities.WorkPosting && d.OpportunityType != entities.InternshipPosting {
		return invalid("opportunity_type must be work or internship")
	}
	if strings.TrimSpace(d.Category) == "" || utf8.RuneCountInString(d.Category) > 200 {
		return invalid("category is required, maximum 200 characters")
	}
	if len(d.Skills) > 30 {
		return invalid("maximum 30 skills")
	}
	if d.SkillLevel != "" && !d.SkillLevel.Valid() {
		return invalid("skill_level must be beginner, intermediate or advanced")
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
	if d.MinimumAge >= 18 && (d.AcceptUnder18 || d.RequiresParentApproval) {
		return invalid("under-18 settings require an age range that includes applicants under 18")
	}
	apply, err := time.Parse("2006-01-02", d.ApplyDeadline)
	if err != nil {
		return invalid("apply_deadline must use YYYY-MM-DD")
	}
	if d.OpportunityType == entities.WorkPosting {
		if strings.TrimSpace(in.Description) == "" {
			return invalid("description is required for work ads")
		}
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
		if d.DurationDays < 0 {
			return invalid("duration_days must be positive when set")
		}
		for _, values := range [][]string{d.Deliverables, d.DeliveryFormats} {
			if len(values) > 30 {
				return invalid("deliverables and delivery_formats accept up to 30 items each")
			}
			for _, value := range values {
				if strings.TrimSpace(value) == "" || utf8.RuneCountInString(value) > 200 {
					return invalid("deliverables and delivery_formats require nonempty items of at most 200 characters")
				}
			}
		}
	} else {
		i := d.Internship
		if i == nil {
			return invalid("internship terms are required")
		}
		if d.BudgetToman != 0 || d.DeliveryDeadline != "" || d.DurationDays != 0 || len(d.Deliverables) > 0 || len(d.DeliveryFormats) > 0 {
			return invalid("internships use stipend and start date instead of work budget and delivery deadline")
		}
		if strings.TrimSpace(i.LearningOutcomes) == "" || strings.TrimSpace(i.Responsibilities) == "" {
			return invalid("learning_outcomes and responsibilities are required")
		}
		if len(i.Responsibilities) > 50000 {
			return invalid("responsibilities cannot exceed 50000 bytes")
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

func jobPostingDescription(in usecase.SaveJobPosting) string {
	description := strings.TrimSpace(in.Description)
	if description == "" && in.Details.Internship != nil {
		return strings.TrimSpace(in.Details.Internship.Responsibilities)
	}
	return description
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
	posting := &entities.JobPosting{EmployerID: employerID, Title: strings.TrimSpace(in.Title), Description: jobPostingDescription(in), Details: details}
	if err := s.uow.Factory().JobPostingRepository().Create(ctx, posting); err != nil {
		return 0, err
	}
	return posting.ID, nil
}

func (s *EmployerJobPostingService) Get(ctx context.Context, token string, id uint64, kind entities.JobPostingType) (*usecase.JobPostingReview, error) {
	employerID, err := s.verifiedEmployerID(ctx, token)
	if err != nil {
		return nil, err
	}
	posting, err := s.uow.Factory().JobPostingRepository().FindOwned(ctx, id, employerID, kind, false)
	if err != nil {
		return nil, err
	}
	attachments, err := s.attachmentDownloads(ctx, posting.Attachments)
	if err != nil {
		return nil, err
	}
	return &usecase.JobPostingReview{JobPosting: posting, Attachments: attachments}, nil
}

func (s *EmployerJobPostingService) attachmentDownloads(ctx context.Context, files []entities.JobPostingAttachment) ([]usecase.JobPostingAttachmentDownload, error) {
	attachments := make([]usecase.JobPostingAttachmentDownload, 0, len(files))
	for _, file := range files {
		url, err := employerDocumentURL(ctx, s.storage, &valueobjects.StoredObject{Key: file.Key})
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, usecase.JobPostingAttachmentDownload{
			Name: file.Name, ContentType: file.ContentType, Size: file.Size, URL: url,
		})
	}
	return attachments, nil
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
	return s.modifyOwned(ctx, id, employerID, in.Details.OpportunityType, func(repo ports.JobPostingRepository, posting *entities.JobPosting) ([]entities.JobPostingAttachment, error) {
		posting.Title, posting.Description, posting.Details = strings.TrimSpace(in.Title), jobPostingDescription(in), details
		return nil, repo.UpdateOwned(ctx, posting)
	})
}

func (s *EmployerJobPostingService) Delete(ctx context.Context, token string, id uint64, kind entities.JobPostingType) error {
	if id == 0 {
		return domainerrors.ErrInvalidJobPosting
	}
	employerID, err := s.verifiedEmployerID(ctx, token)
	if err != nil {
		return err
	}
	return s.modifyOwned(ctx, id, employerID, kind, func(repo ports.JobPostingRepository, posting *entities.JobPosting) ([]entities.JobPostingAttachment, error) {
		_, err := repo.DeleteOwned(ctx, id, employerID)
		return posting.Attachments, err
	})
}

func (s *EmployerJobPostingService) ReplaceAttachments(ctx context.Context, token string, id uint64, files []ports.UploadFile) error {
	employerID, err := s.verifiedEmployerID(ctx, token)
	if err != nil {
		return err
	}
	if len(files) > usecase.MaxJobPostingAttachments {
		return fmt.Errorf("%w: maximum %d attachments", domainerrors.ErrInvalidUpload, usecase.MaxJobPostingAttachments)
	}
	validated := make([]*ports.ValidatedFile, 0, len(files))
	for _, file := range files {
		result, err := s.media.JobPostingAttachment(file, usecase.MaxJobPostingAttachmentBytes)
		if err != nil {
			return err
		}
		validated = append(validated, result)
	}
	uploaded := make([]string, 0, len(validated))
	err = s.modifyOwned(ctx, id, employerID, entities.WorkPosting, func(repo ports.JobPostingRepository, posting *entities.JobPosting) ([]entities.JobPostingAttachment, error) {
		attachments := make([]entities.JobPostingAttachment, 0, len(validated))
		for _, file := range validated {
			var nonce [16]byte
			if _, err := rand.Read(nonce[:]); err != nil {
				return nil, err
			}
			key := "job-postings/" + hex.EncodeToString(nonce[:]) + file.Extension
			uploaded = append(uploaded, key)
			object, err := s.storage.Put(ctx, key, file.Body, file.ContentType, file.Size)
			if err != nil {
				return nil, domainerrors.ErrStorageUnavailable
			}
			attachments = append(attachments, entities.JobPostingAttachment{
				Name: filepath.Base(file.Name), Key: object.Key, ContentType: object.ContentType, Size: object.Size,
			})
		}
		removed := posting.Attachments
		posting.Attachments = attachments
		return removed, repo.UpdateOwned(ctx, posting)
	})
	if err != nil {
		cleanupEmployerUploads(ctx, s.storage, uploaded)
	}
	return err
}

func (s *EmployerJobPostingService) modifyOwned(ctx context.Context, id, employerID uint64, kind entities.JobPostingType, modify func(ports.JobPostingRepository, *entities.JobPosting) ([]entities.JobPostingAttachment, error)) error {
	var removed []entities.JobPostingAttachment
	err := s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobPostingRepository()
		posting, err := repo.FindOwned(ctx, id, employerID, kind, true)
		if err != nil {
			return err
		}
		removed, err = modify(repo, posting)
		return err
	})
	if err == nil && len(removed) > 0 {
		keys := make([]string, 0, len(removed))
		for _, file := range removed {
			keys = append(keys, file.Key)
		}
		cleanupEmployerUploads(ctx, s.storage, keys)
	}
	return err
}
