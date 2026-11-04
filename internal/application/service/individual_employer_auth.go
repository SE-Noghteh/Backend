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

func (s *IndividualEmployerService) authentication() employerAuthentication[entities.IndividualEmployer] {
	return employerAuthentication[entities.IndividualEmployer]{uow: s.uow, passwords: s.passwords, tokens: s.tokens, role: enums.RoleIndividualEmployer,
		findPhone: func(ctx context.Context, r ports.EmployerRepository, phone string) (*entities.IndividualEmployer, error) {
			return r.FindIndividualByPhone(ctx, phone)
		},
		findID: func(ctx context.Context, r ports.EmployerRepository, id uint64) (*entities.IndividualEmployer, error) {
			return r.FindIndividualByID(ctx, id)
		},
		credentials: func(e *entities.IndividualEmployer) (uint64, string, bool) {
			return e.Employer.ID, e.PasswordHash, !e.PhoneVerifiedAt.IsZero()
		},
	}
}

func (s *IndividualEmployerService) Login(ctx context.Context, phone, password string) (*usecase.IndividualEmployerLoginResult, error) {
	e, token, err := s.authentication().login(ctx, phone, password)
	if err != nil {
		return nil, err
	}
	return &usecase.IndividualEmployerLoginResult{Employer: e, AccessToken: token}, nil
}

func (s *IndividualEmployerService) Authenticate(ctx context.Context, token string) (*entities.IndividualEmployer, error) {
	return s.authentication().authenticate(ctx, token)
}
func (s *IndividualEmployerService) Logout(ctx context.Context, token string) error {
	return revokeToken(ctx, s.uow.Factory(), s.tokens, token, enums.RoleIndividualEmployer)
}

func (s *IndividualEmployerService) DocumentURL(ctx context.Context, token, kind string) (string, error) {
	employer, err := s.Authenticate(ctx, token)
	if err != nil {
		return "", err
	}
	var object *valueobjects.StoredObject
	switch kind {
	case "identity":
		object = &employer.IdentityDocument
	case "video":
		object = &employer.VerificationVideo
	case "business-license":
		object = employer.BusinessLicense
	default:
		return "", fmt.Errorf("%w: document kind", domainerrors.ErrInvalidSignup)
	}
	if object == nil {
		return "", fmt.Errorf("%w: no business license was uploaded", domainerrors.ErrInvalidSignup)
	}
	return employerDocumentURL(ctx, s.storage, object)
}
