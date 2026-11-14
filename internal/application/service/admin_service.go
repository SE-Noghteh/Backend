package service

import (
	"context"
	"fmt"
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

type AdminService struct {
	uow       ports.UnitOfWork
	passwords ports.PasswordHasher
	tokens    ports.AdminTokenManager
	storage   ports.Storage
}

func NewAdminService(uow ports.UnitOfWork, passwords ports.PasswordHasher, tokens ports.AdminTokenManager, storage ports.Storage) *AdminService {
	return &AdminService{uow: uow, passwords: passwords, tokens: tokens, storage: storage}
}
func (s *AdminService) Login(ctx context.Context, username, password string) (*usecase.AdminLoginResult, error) {
	username = strings.TrimSpace(username)
	if username == "" || len(username) > 100 || len(password) == 0 || len(password) > 72 {
		return nil, domainerrors.ErrInvalidAdminCredentials
	}
	var result *usecase.AdminLoginResult
	var authErr error
	err := s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		repo := f.AdminRepository()
		if err := repo.LockUsername(ctx, username); err != nil {
			return err
		}
		attempt, err := nextLoginAttempt(ctx, repo, username)
		if err != nil {
			return err
		}
		admin, err := repo.FindByUsername(ctx, username)
		if err != nil {
			return err
		}
		hash := ""
		if admin != nil {
			hash = admin.PasswordHash
		}
		if !s.passwords.Matches(hash, password) || admin == nil {
			attempt.Failures++
			authErr = domainerrors.ErrInvalidAdminCredentials
			return repo.SaveLoginAttempt(ctx, attempt)
		}
		token, err := s.tokens.IssueAdminAccessToken(admin.Username)
		if err != nil {
			return err
		}
		if err := repo.DeleteLoginAttempt(ctx, username); err != nil {
			return err
		}
		result = &usecase.AdminLoginResult{Username: admin.Username, AccessToken: token}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if authErr != nil {
		return nil, authErr
	}
	return result, nil
}
func (s *AdminService) Authenticate(ctx context.Context, token string) (string, error) {
	username, err := s.tokens.VerifyAdminAccessToken(token)
	if err != nil {
		return "", domainerrors.ErrUnauthenticated
	}
	admin, err := s.uow.Factory().AdminRepository().FindByUsername(ctx, username)
	if err != nil {
		return "", err
	}
	if admin == nil {
		return "", domainerrors.ErrUnauthenticated
	}
	return username, nil
}
func adminListFilter(f usecase.AdminListFilter) (usecase.AdminListFilter, error) {
	if f.Status == "" {
		f.Status = "pending"
	}
	if f.Limit == 0 {
		f.Limit = 20
	}
	if f.Limit < 1 || f.Limit > 100 || f.Offset < 0 {
		return f, fmt.Errorf("%w: limit must be 1-100 and offset nonnegative", domainerrors.ErrInvalidReview)
	}
	switch f.Status {
	case "pending", "approved", "rejected", "all":
	default:
		return f, fmt.Errorf("%w: unsupported status filter", domainerrors.ErrInvalidReview)
	}
	return f, nil
}
func (s *AdminService) JobSeekers(ctx context.Context, in usecase.AdminListFilter) ([]*entities.JobSeeker, int64, error) {
	f, err := adminListFilter(in)
	if err != nil {
		return nil, 0, err
	}
	return s.uow.Factory().AdminReviewRepository().ListJobSeekers(ctx, f.Status, f.Limit, f.Offset)
}
func (s *AdminService) Parents(ctx context.Context, in usecase.AdminListFilter) ([]*entities.Parent, int64, error) {
	f, err := adminListFilter(in)
	if err != nil {
		return nil, 0, err
	}
	return s.uow.Factory().AdminReviewRepository().ListParents(ctx, f.Status, f.Limit, f.Offset)
}
func (s *AdminService) JobSeeker(ctx context.Context, id uint64) (*entities.JobSeeker, error) {
	p, err := s.uow.Factory().AdminReviewRepository().FindJobSeeker(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domainerrors.ErrReviewNotFound
	}
	return p, nil
}
func (s *AdminService) Parent(ctx context.Context, id uint64) (*entities.Parent, error) {
	p, err := s.uow.Factory().AdminReviewRepository().FindParent(ctx, id)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, domainerrors.ErrReviewNotFound
	}
	return p, nil
}
func (s *AdminService) JobSeekerVideoURL(ctx context.Context, id uint64) (string, error) {
	p, err := s.JobSeeker(ctx, id)
	if err != nil {
		return "", err
	}
	return employerDocumentURL(ctx, s.storage, p.VerificationVideo)
}
func (s *AdminService) ParentDocumentURL(ctx context.Context, id uint64, kind string) (string, error) {
	p, err := s.Parent(ctx, id)
	if err != nil {
		return "", err
	}
	switch kind {
	case "identity":
		return employerDocumentURL(ctx, s.storage, p.IdentityDocument)
	case "consent-video":
		return employerDocumentURL(ctx, s.storage, p.ConsentVideo)
	default:
		return "", fmt.Errorf("%w: document kind must be identity or consent-video", domainerrors.ErrInvalidReview)
	}
}
func newAdminReview(in usecase.AdminReviewDecision, target string) (valueobjects.AdminReview, error) {
	message := strings.TrimSpace(in.Message)
	if utf8.RuneCountInString(message) > 1000 || in.Reviewer == "" {
		return valueobjects.AdminReview{}, fmt.Errorf("%w: message must be at most 1000 characters", domainerrors.ErrInvalidReview)
	}
	now := time.Now().UTC()
	status := enums.AdminReviewApproved
	if !in.Approved {
		status = enums.AdminReviewRejected
		if message == "" {
			message = target + " was not approved"
		}
	}
	return valueobjects.AdminReview{AdminApprovalStatus: status, AdminReviewMessage: message, AdminReviewedAt: &now, AdminReviewedBy: in.Reviewer}, nil
}
func (s *AdminService) ReviewJobSeeker(ctx context.Context, id uint64, in usecase.AdminReviewDecision) (*entities.JobSeeker, error) {
	review, err := newAdminReview(in, "Job seeker")
	if err != nil {
		return nil, err
	}
	var result *entities.JobSeeker
	err = s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		admin, err := f.AdminRepository().FindByUsername(ctx, in.Reviewer)
		if err != nil {
			return err
		}
		if admin == nil {
			return domainerrors.ErrUnauthenticated
		}
		repo := f.AdminReviewRepository()
		seeker, err := repo.FindJobSeeker(ctx, id)
		if err != nil {
			return err
		}
		if seeker == nil {
			return domainerrors.ErrReviewNotFound
		}
		if err := f.JobSeekerRepository().LockAccount(ctx, id); err != nil {
			return err
		}
		seeker, err = repo.FindJobSeeker(ctx, id)
		if err != nil {
			return err
		}
		if seeker == nil {
			return domainerrors.ErrReviewNotFound
		}
		age, err := jobSeekerAge(seeker, time.Now())
		if err != nil {
			return err
		}
		if age < 18 && seeker.ParentID == nil {
			return domainerrors.ErrSignupIncomplete
		}
		if err := repo.ReviewJobSeeker(ctx, id, review); err != nil {
			return err
		}
		result, err = repo.FindJobSeeker(ctx, id)
		return err
	})
	return result, err
}
func (s *AdminService) ReviewParent(ctx context.Context, id uint64, in usecase.AdminReviewDecision) (*entities.Parent, error) {
	review, err := newAdminReview(in, "Parent")
	if err != nil {
		return nil, err
	}
	var result *entities.Parent
	err = s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		admin, err := f.AdminRepository().FindByUsername(ctx, in.Reviewer)
		if err != nil {
			return err
		}
		if admin == nil {
			return domainerrors.ErrUnauthenticated
		}
		repo := f.AdminReviewRepository()
		parent, err := repo.FindParent(ctx, id)
		if err != nil {
			return err
		}
		if parent == nil {
			return domainerrors.ErrReviewNotFound
		}
		if err := repo.ReviewParent(ctx, id, review); err != nil {
			return err
		}
		result, err = repo.FindParent(ctx, id)
		return err
	})
	return result, err
}
