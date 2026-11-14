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

func (s *LegalEmployerService) authentication() employerAuthentication[entities.LegalEmployer] {
	return employerAuthentication[entities.LegalEmployer]{uow: s.uow, passwords: s.passwords, tokens: s.tokens, role: enums.RoleLegalEmployer,
		findPhone: func(ctx context.Context, r ports.EmployerRepository, phone string) (*entities.LegalEmployer, error) {
			return r.FindLegalByPhone(ctx, phone)
		},
		findID: func(ctx context.Context, r ports.EmployerRepository, id uint64) (*entities.LegalEmployer, error) {
			return r.FindLegalByID(ctx, id)
		},
		credentials: func(e *entities.LegalEmployer) (uint64, string, bool) {
			return e.Employer.ID, e.PasswordHash, !e.PhoneVerifiedAt.IsZero()
		},
	}
}

func (s *LegalEmployerService) Login(ctx context.Context, phone, password string) (*usecase.LegalEmployerLoginResult, error) {
	e, token, err := s.authentication().login(ctx, phone, password)
	if err != nil {
		return nil, err
	}
	return &usecase.LegalEmployerLoginResult{Employer: e, AccessToken: token}, nil
}

func (s *LegalEmployerService) Authenticate(ctx context.Context, token string) (*entities.LegalEmployer, error) {
	return s.authentication().authenticate(ctx, token)
}
func (s *LegalEmployerService) Logout(ctx context.Context, token string) error {
	return revokeToken(ctx, s.uow.Factory(), s.tokens, token, enums.RoleLegalEmployer)
}

func (s *LegalEmployerService) DocumentURL(ctx context.Context, token, kind string) (string, error) {
	e, err := s.Authenticate(ctx, token)
	if err != nil {
		return "", err
	}
	var object *valueobjects.StoredObject
	switch kind {
	case "establishment":
		object = &e.EstablishmentDocument
	case "identity":
		object = &e.IdentityDocument
	case "video":
		object = &e.VerificationVideo
	case "activity-license":
		object = e.ActivityLicense
	case "authorization":
		object = e.AuthorizationDocument
	default:
		return "", fmt.Errorf("%w: document kind", domainerrors.ErrInvalidSignup)
	}
	if object == nil {
		return "", fmt.Errorf("%w: no document was uploaded for this kind", domainerrors.ErrInvalidSignup)
	}
	return employerDocumentURL(ctx, s.storage, object)
}
