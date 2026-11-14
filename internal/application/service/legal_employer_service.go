package service

import (
	"context"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

type LegalEmployerService struct {
	uow       ports.UnitOfWork
	sender    ports.OTPSender
	passwords ports.PasswordHasher
	tokens    ports.TokenManager
	storage   ports.Storage
	media     ports.MediaValidator
}

func NewLegalEmployerService(uow ports.UnitOfWork, sender ports.OTPSender, passwords ports.PasswordHasher, tokens ports.TokenManager, storage ports.Storage, media ports.MediaValidator) *LegalEmployerService {
	return &LegalEmployerService{uow: uow, sender: sender, passwords: passwords, tokens: tokens, storage: storage, media: media}
}

func (s *LegalEmployerService) Signup(ctx context.Context, in usecase.SignupLegalEmployer) error {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return err
	}
	if err := validateLegalEmployer(&in); err != nil {
		return err
	}
	var uploaded, replaced []string
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.EmployerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		exists, err := repo.LegalExists(ctx, phone, in.CompanyNationalID)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrSignupConflict
		}
		signups := factory.LegalEmployerSignupRepository()
		previous, err := signups.GetSignup(ctx, phone)
		if err != nil {
			return err
		}
		if previous != nil && time.Since(previous.Challenge.SentAt) < otpResendDelay {
			return domainerrors.ErrOTPTooSoon
		}
		hash, err := s.passwords.Hash(in.Password)
		if err != nil {
			return err
		}
		objects, err := uploadEmployerFiles(ctx, s.storage, s.media, "legal-employers", []employerUpload{
			{kind: "establishment", file: &in.EstablishmentDocument},
			{kind: "identity", file: &in.IdentityDocument},
			{kind: "video", file: &in.VerificationVideo, video: true},
			{kind: "activity-license", file: in.ActivityLicense},
			{kind: "authorization", file: in.AuthorizationDocument},
		}, &uploaded)
		if err != nil {
			return err
		}
		registration := &entities.LegalEmployer{
			Employer:    entities.Employer{Type: enums.EmployerLegal},
			CompanyName: in.CompanyName, LegalEntityType: in.LegalEntityType, CompanyNationalID: in.CompanyNationalID,
			RegistrationNumber: in.RegistrationNumber, EconomicCode: in.EconomicCode,
			Province: in.Province, City: in.City, Address: in.Address, PostalCode: in.PostalCode,
			RepresentativeFirstName: in.RepresentativeFirstName, RepresentativeLastName: in.RepresentativeLastName,
			RepresentativePosition: in.RepresentativePosition, RepresentativeGender: in.RepresentativeGender, RepresentativeNationalCode: in.RepresentativeNationalCode,
			HasTaxFile: in.HasTaxFile, Activities: in.Activities, Website: optionalString(in.Website), RequiresActivityLicense: in.RequiresActivityLicense,
			Phone: phone, Email: optionalString(in.Email), PasswordHash: hash,
			EstablishmentDocument: *objects["establishment"], IdentityDocument: *objects["identity"], VerificationVideo: *objects["video"],
			ActivityLicense: objects["activity-license"], AuthorizationDocument: objects["authorization"],
			CreatedAt: time.Now().UTC(), ReviewStatus: enums.EmployerPendingReview,
		}
		pending := &entities.PendingLegalEmployerSignup{Registration: registration}
		if err := sendEmployerChallenge(ctx, s.sender, signups, phone, pending); err != nil {
			return err
		}
		if previous != nil {
			replaced = legalEmployerObjectKeys(previous.Registration)
		}
		return nil
	})
	if err != nil {
		cleanupEmployerUploads(ctx, s.storage, uploaded)
		return err
	}
	cleanupEmployerUploads(ctx, s.storage, replaced)
	return nil
}

func legalEmployerObjectKeys(e *entities.LegalEmployer) []string {
	if e == nil {
		return nil
	}
	keys := []string{e.EstablishmentDocument.Key, e.IdentityDocument.Key, e.VerificationVideo.Key}
	if e.ActivityLicense != nil {
		keys = append(keys, e.ActivityLicense.Key)
	}
	if e.AuthorizationDocument != nil {
		keys = append(keys, e.AuthorizationDocument.Key)
	}
	return keys
}

func (s *LegalEmployerService) signupFlow() employerSignupFlow[entities.LegalEmployer] {
	return employerSignupFlow[entities.LegalEmployer]{uow: s.uow, sender: s.sender,
		repository: func(f ports.RepositoryFactory) ports.LegalEmployerSignupRepository {
			return f.LegalEmployerSignupRepository()
		},
		createdAt: func(e *entities.LegalEmployer) time.Time { return e.CreatedAt },
		finalize: func(ctx context.Context, f ports.RepositoryFactory, e *entities.LegalEmployer) error {
			repo := f.EmployerRepository()
			exists, err := repo.LegalExists(ctx, e.Phone, e.CompanyNationalID)
			if err != nil {
				return err
			}
			if exists {
				return domainerrors.ErrSignupConflict
			}
			now := time.Now().UTC()
			e.PhoneVerifiedAt, e.CreatedAt = now, now
			return repo.CreateLegal(ctx, e)
		},
	}
}

func (s *LegalEmployerService) RequestOTP(ctx context.Context, phone string) error {
	return s.signupFlow().resend(ctx, phone)
}
func (s *LegalEmployerService) VerifyOTP(ctx context.Context, in usecase.VerifySignupOTP) (*entities.LegalEmployer, error) {
	return s.signupFlow().verify(ctx, in.Phone, in.OTP)
}
