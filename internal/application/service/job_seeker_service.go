package service

import (
	"context"
	"crypto/subtle"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

type JobSeekerService struct {
	uow       ports.UnitOfWork
	sender    ports.OTPSender
	passwords ports.PasswordHasher
	storage   ports.Storage
	media     ports.MediaValidator
}

func NewJobSeekerService(uow ports.UnitOfWork, sender ports.OTPSender, passwords ports.PasswordHasher, storage ports.Storage, media ports.MediaValidator) *JobSeekerService {
	return &JobSeekerService{uow: uow, sender: sender, passwords: passwords, storage: storage, media: media}
}

// Signup validates the entire profile before storing a pending registration or sending SMS.
// An account becomes visible to login only after VerifyOTP succeeds.
func (s *JobSeekerService) Signup(ctx context.Context, in usecase.SignupJobSeeker) error {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return err
	}
	if err := validateSignup(&in); err != nil {
		return err
	}
	if in.VerificationVideo.Body == nil {
		return domainerrors.ErrInvalidUpload
	}
	hash, err := s.passwords.Hash(in.Password)
	if err != nil {
		return err
	}
	registration := &entities.JobSeeker{
		FirstName:    in.FirstName,
		LastName:     in.LastName,
		NationalCode: in.NationalCode,
		City:         in.City,
		Address:      in.Address,
		BirthDate:    in.BirthDate,
		Gender:       in.Gender,
		Interests:    in.Interests,
		Phone:        phone,
		PasswordHash: hash,
		CreatedAt:    time.Now().UTC(),
	}
	var uploaded, replaced []string
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		exists, err := repo.Exists(ctx, phone, in.NationalCode)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrSignupConflict
		}
		otpRepo := factory.SignupOTPRepository()
		previous, err := otpRepo.GetOTP(ctx, phone)
		if err != nil {
			return err
		}
		if previous != nil && time.Since(previous.SentAt) < otpResendDelay {
			return domainerrors.ErrOTPTooSoon
		}
		objects, err := uploadEmployerFiles(ctx, s.storage, s.media, "job-seekers", []employerUpload{{kind: "video", file: &in.VerificationVideo, video: true}}, &uploaded)
		if err != nil {
			return err
		}
		registration.VerificationVideo = objects["video"]
		if err := s.sendChallenge(ctx, otpRepo, registration, previous); err != nil {
			return err
		}
		if previous != nil && previous.Registration != nil && previous.Registration.VerificationVideo != nil {
			replaced = append(replaced, previous.Registration.VerificationVideo.Key)
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

// RequestOTP resends only for a previously validated, unexpired pending profile.
func (s *JobSeekerService) RequestOTP(ctx context.Context, rawPhone string) error {
	phone, err := normalizePhone(rawPhone)
	if err != nil {
		return err
	}
	return s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		otpRepo := factory.SignupOTPRepository()
		previous, err := otpRepo.GetOTP(ctx, phone)
		if err != nil {
			return err
		}
		if !pendingSignupValid(previous, time.Now().UTC()) {
			return domainerrors.ErrNoPendingSignup
		}
		return s.sendChallenge(ctx, otpRepo, previous.Registration, previous)
	})
}

func pendingSignupValid(otp *entities.SignupOTP, now time.Time) bool {
	return otp != nil && otp.Registration != nil && otp.Registration.VerificationVideo != nil && otp.Registration.VerificationVideo.Key != "" && now.Before(otp.Registration.CreatedAt.Add(pendingSignupLifetime))
}

func (s *JobSeekerService) VideoURL(ctx context.Context, id uint64) (string, error) {
	seeker, err := s.uow.Factory().JobSeekerRepository().FindByID(ctx, id)
	if err != nil {
		return "", err
	}
	if seeker == nil {
		return "", domainerrors.ErrUnauthenticated
	}
	return employerDocumentURL(ctx, s.storage, seeker.VerificationVideo)
}

func (s *JobSeekerService) sendChallenge(ctx context.Context, repo ports.SignupOTPRepository, registration *entities.JobSeeker, previous *entities.SignupOTP) error {
	now := time.Now().UTC()
	if previous != nil && now.Sub(previous.SentAt) < otpResendDelay {
		return domainerrors.ErrOTPTooSoon
	}
	challenge, code, err := newSignupChallenge(registration.Phone, now)
	if err != nil {
		return err
	}
	challenge.Registration = registration
	if err := repo.SaveOTP(ctx, challenge); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, "98"+registration.Phone[1:], code); err != nil {
		return domainerrors.ErrSMSUnavailable
	}
	return nil
}

func (s *JobSeekerService) VerifyOTP(ctx context.Context, in usecase.VerifySignupOTP) (*entities.JobSeeker, error) {
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return nil, err
	}
	var result *entities.JobSeeker
	var verificationErr error
	err = s.uow.WithTransaction(func(factory ports.RepositoryFactory) error {
		repo := factory.JobSeekerRepository()
		if err := repo.LockPhone(ctx, phone); err != nil {
			return err
		}
		otpRepo := factory.SignupOTPRepository()
		otp, err := otpRepo.GetOTP(ctx, phone)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if !pendingSignupValid(otp, now) || !now.Before(otp.ExpiresAt) || otp.Attempts >= maxOTPAttempts {
			verificationErr = domainerrors.ErrInvalidOTP
			return nil
		}
		actual := otpHash(otp.Salt, in.OTP)
		if !codePattern.MatchString(in.OTP) || subtle.ConstantTimeCompare([]byte(actual), []byte(otp.CodeHash)) != 1 {
			otp.Attempts++
			verificationErr = domainerrors.ErrInvalidOTP
			return otpRepo.SaveOTP(ctx, otp) // Commit failed attempts rather than rolling them back.
		}
		exists, err := repo.Exists(ctx, phone, otp.Registration.NationalCode)
		if err != nil {
			return err
		}
		if exists {
			return domainerrors.ErrSignupConflict
		}
		result = otp.Registration
		result.SignupStatus = entities.SignupFirstStageCompleted
		result.PhoneVerifiedAt, result.CreatedAt = now, now
		if err := repo.Create(ctx, result); err != nil {
			return err
		}
		return otpRepo.DeleteOTP(ctx, phone)
	})
	if err != nil {
		return nil, err
	}
	if verificationErr != nil {
		return nil, verificationErr
	}
	return result, nil
}
