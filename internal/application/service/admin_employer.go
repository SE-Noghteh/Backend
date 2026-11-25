package service

import (
	"context"
	"fmt"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

func (s *AdminService) Employers(ctx context.Context, in usecase.AdminEmployerListFilter) ([]entities.EmployerReviewSummary, int64, error) {
	if in.Status == string(enums.EmployerPendingReview) {
		in.Status = "pending"
	}
	filter, err := adminListFilter(in.AdminListFilter)
	if err != nil {
		return nil, 0, err
	}
	if filter.Status == "pending" {
		filter.Status = string(enums.EmployerPendingReview)
	}
	switch in.Type {
	case "all":
		in.Type = ""
	case "", enums.EmployerIndividual, enums.EmployerLegal:
	default:
		return nil, 0, fmt.Errorf("%w: type must be individual, legal, or all", domainerrors.ErrInvalidReview)
	}
	return s.uow.Factory().AdminReviewRepository().ListEmployers(ctx, in.Type, filter.Status, filter.Limit, filter.Offset)
}

func (s *AdminService) Employer(ctx context.Context, id uint64) (*entities.EmployerReviewProfile, error) {
	return employerForReview(ctx, s.uow.Factory().AdminReviewRepository(), id)
}

func employerForReview(ctx context.Context, repo ports.AdminReviewRepository, id uint64) (*entities.EmployerReviewProfile, error) {
	profile, err := repo.FindEmployer(ctx, id)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		return nil, domainerrors.ErrReviewNotFound
	}
	return profile, nil
}

func (s *AdminService) EmployerDocumentURL(ctx context.Context, id uint64, kind string) (string, error) {
	profile, err := s.Employer(ctx, id)
	if err != nil {
		return "", err
	}
	var file *valueobjects.StoredObject
	if profile.Individual != nil {
		file, err = individualEmployerDocument(profile.Individual, kind)
	} else {
		file, err = legalEmployerDocument(profile.Legal, kind)
	}
	if err != nil {
		return "", fmt.Errorf("%w: unsupported document kind for this employer type", domainerrors.ErrInvalidReview)
	}
	if file == nil || file.Key == "" {
		return "", domainerrors.ErrReviewDocumentNotFound
	}
	return employerDocumentURL(ctx, s.storage, file)
}

func (s *AdminService) ReviewEmployer(ctx context.Context, id uint64, in usecase.AdminReviewDecision) (*entities.EmployerReviewProfile, error) {
	review, err := newAdminReview(in, "Employer")
	if err != nil {
		return nil, err
	}
	var result *entities.EmployerReviewProfile
	err = s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		admin, err := f.AdminRepository().FindByUsername(ctx, in.Reviewer)
		if err != nil {
			return err
		}
		if admin == nil {
			return domainerrors.ErrUnauthenticated
		}
		repo := f.AdminReviewRepository()
		profile, err := employerForReview(ctx, repo, id)
		if err != nil {
			return err
		}
		kind := enums.EmployerLegal
		if profile.Individual != nil {
			kind = enums.EmployerIndividual
		}
		if err := repo.ReviewEmployer(ctx, id, kind, review); err != nil {
			return err
		}
		result, err = employerForReview(ctx, repo, id)
		return err
	})
	return result, err
}
