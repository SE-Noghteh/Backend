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

type IndividualEmployerService struct {
	uow       ports.UnitOfWork
	sender    ports.OTPSender
	passwords ports.PasswordHasher
	tokens    ports.TokenManager
	storage   ports.Storage
	media     ports.MediaValidator
}

func NewIndividualEmployerService(uow ports.UnitOfWork, sender ports.OTPSender, passwords ports.PasswordHasher, tokens ports.TokenManager, storage ports.Storage, media ports.MediaValidator) *IndividualEmployerService {
	return &IndividualEmployerService{uow: uow, sender: sender, passwords: passwords, tokens: tokens, storage: storage, media: media}
}

// Signup saves the complete validated submission and sends OTP automatically.
// The employer account is created only by VerifyOTP.
func (s *IndividualEmployerService) Signup(ctx context.Context, in usecase.SignupIndividualEmployer) error {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return err
	}
	if err := validateIndividualEmployer(&in); err != nil {
		return err
	}
	var uploaded, replaced []string
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.EmployerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		exists, err := repo.IndividualExists(ctx, phone, in.NationalCode)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrSignupConflict
		}
		signups := factory.EmployerSignupRepository()
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
		documents, err := s.uploadSignupDocuments(ctx, in, &uploaded)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		registration := &entities.IndividualEmployer{
			Employer:  entities.Employer{Type: enums.EmployerIndividual},
			FirstName: in.FirstName, LastName: in.LastName, FatherName: in.FatherName, NationalCode: in.NationalCode,
			Gender: in.Gender, BirthDate: in.BirthDate, Province: in.Province, City: in.City, Address: in.Address, PostalCode: in.PostalCode,
			BusinessName: optionalString(in.BusinessName), BusinessActivity: optionalString(in.BusinessActivity), Website: optionalString(in.Website), HasLegalGuardian: in.HasLegalGuardian,
			Phone: phone, Email: optionalString(in.Email), PasswordHash: hash,
			IdentityDocument: documents.identity, BusinessLicense: documents.license, VerificationVideo: documents.video,
			CreatedAt: now, ReviewStatus: enums.EmployerPendingReview,
		}
		pending := &entities.PendingIndividualEmployerSignup{Registration: registration}
		if err := sendEmployerChallenge(ctx, s.sender, signups, phone, pending); err != nil {
			return err
		}
		if previous != nil {
			replaced = employerObjectKeys(previous.Registration)
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

func (s *IndividualEmployerService) signupFlow() employerSignupFlow[entities.IndividualEmployer] {
	return employerSignupFlow[entities.IndividualEmployer]{uow: s.uow, sender: s.sender,
		repository: func(f ports.RepositoryFactory) ports.EmployerSignupRepository { return f.EmployerSignupRepository() },
		createdAt:  func(e *entities.IndividualEmployer) time.Time { return e.CreatedAt },
		finalize: func(ctx context.Context, f ports.RepositoryFactory, e *entities.IndividualEmployer) error {
			repo := f.EmployerRepository()
			exists, err := repo.IndividualExists(ctx, e.Phone, e.NationalCode)
			if err != nil {
				return err
			}
			if exists {
				return domainerrors.ErrSignupConflict
			}
			now := time.Now().UTC()
			e.PhoneVerifiedAt, e.CreatedAt = now, now
			return repo.CreateIndividual(ctx, e)
		}}
}

// RequestOTP only resends for a previously validated pending signup.
func (s *IndividualEmployerService) RequestOTP(ctx context.Context, phone string) error {
	return s.signupFlow().resend(ctx, phone)
}

func (s *IndividualEmployerService) VerifyOTP(ctx context.Context, in usecase.VerifySignupOTP) (*entities.IndividualEmployer, error) {
	return s.signupFlow().verify(ctx, in.Phone, in.OTP)
}
