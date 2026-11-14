package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
)

const parentInvitationLifetime = 7 * 24 * time.Hour

type ParentService struct {
	uow       ports.UnitOfWork
	sender    ports.OTPSender
	passwords ports.PasswordHasher
	storage   ports.Storage
	media     ports.MediaValidator
}

func NewParentService(uow ports.UnitOfWork, sender ports.OTPSender, passwords ports.PasswordHasher, storage ports.Storage, media ports.MediaValidator) *ParentService {
	return &ParentService{uow: uow, sender: sender, passwords: passwords, storage: storage, media: media}
}
func invitationHash(code string) string {
	hash := sha256.Sum256([]byte(code))
	return hex.EncodeToString(hash[:])
}
func invalidParent(message string) error {
	return fmt.Errorf("%w: %s", domainerrors.ErrInvalidParent, message)
}
func jobSeekerAge(seeker *entities.JobSeeker, now time.Time) (int, error) {
	birth, err := seeker.BirthDate.Gregorian()
	if err != nil {
		return 0, err
	}
	now = now.UTC()
	age := now.Year() - birth.Year()
	if now.Before(birth.AddDate(age, 0, 0)) {
		age--
	}
	return age, nil
}

func (s *ParentService) SetJobSeekerParent(ctx context.Context, id uint64, in usecase.JobSeekerParentInput) (*usecase.ParentStageResult, error) {
	var phone string
	if !in.Skip {
		in.FirstName, in.LastName = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName)
		if in.FirstName == "" || in.LastName == "" || utf8.RuneCountInString(in.FirstName) > 100 || utf8.RuneCountInString(in.LastName) > 100 {
			return nil, invalidParent("parent first_name and last_name are required and must be at most 100 characters")
		}
		var err error
		phone, err = normalizePhone(in.Phone)
		if err != nil {
			return nil, err
		}
	}
	var result *usecase.ParentStageResult
	err := s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		parents := f.ParentRepository()
		// Parent-phone locks always precede account locks, across invitation and OTP operations.
		if !in.Skip {
			if err := parents.LockPhone(ctx, phone); err != nil {
				return err
			}
		}
		seekers := f.JobSeekerRepository()
		if err := seekers.LockAccount(ctx, id); err != nil {
			return err
		}
		seeker, err := seekers.FindByID(ctx, id)
		if err != nil {
			return err
		}
		if seeker == nil || seeker.PhoneVerifiedAt.IsZero() {
			return domainerrors.ErrUnauthenticated
		}
		if in.Skip {
			age, err := jobSeekerAge(seeker, time.Now())
			if err != nil {
				return err
			}
			if age < 18 {
				return domainerrors.ErrParentRequired
			}
			if err := parents.DeleteInvitation(ctx, id); err != nil {
				return err
			}
			if err := parents.SetSignupStage(ctx, id, entities.SignupSecondStageCompleted, seeker.ParentID, seeker.ParentRelationship); err != nil {
				return err
			}
			result = &usecase.ParentStageResult{SignupStatus: entities.SignupSecondStageCompleted, ParentID: seeker.ParentID}
			return nil
		}
		if phone == seeker.Phone {
			return invalidParent("parent phone must differ from job seeker phone")
		}
		parent, err := parents.FindByPhone(ctx, phone)
		if err != nil {
			return err
		}
		if parent != nil && !parent.PhoneVerifiedAt.IsZero() {
			if err := parents.SetSignupStage(ctx, id, entities.SignupSecondStageCompleted, &parent.ID, parent.Relationship); err != nil {
				return err
			}
			if err := parents.DeleteInvitation(ctx, id); err != nil {
				return err
			}
			result = &usecase.ParentStageResult{SignupStatus: entities.SignupSecondStageCompleted, ParentID: &parent.ID}
			return nil
		}
		if seeker.ParentID != nil {
			return invalidParent("a parent is already linked to this job seeker")
		}
		nonce := make([]byte, 32)
		if _, err := rand.Read(nonce); err != nil {
			return err
		}
		code := base64.RawURLEncoding.EncodeToString(nonce)
		expires := time.Now().UTC().Add(parentInvitationLifetime)
		invitation := &entities.ParentInvitation{JobSeekerID: id, CodeHash: invitationHash(code), FirstName: in.FirstName, LastName: in.LastName, Phone: phone, ExpiresAt: expires}
		if err := parents.SaveInvitation(ctx, invitation); err != nil {
			return err
		}
		if err := parents.SetSignupStage(ctx, id, entities.SignupFirstStageCompleted, nil, ""); err != nil {
			return err
		}
		result = &usecase.ParentStageResult{SignupStatus: entities.SignupFirstStageCompleted, InvitationCode: code, ExpiresAt: &expires}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Check and lock the invitation's account after acquiring the parent phone lock.
// Re-read after locking so replacement, skipping, or deletion invalidates old codes.
func lockedParentInvitation(ctx context.Context, f ports.RepositoryFactory, code string) (*entities.ParentInvitation, error) {
	if len(code) != 43 {
		return nil, domainerrors.ErrInvalidInvitation
	}
	repo := f.ParentRepository()
	i, err := repo.FindInvitation(ctx, invitationHash(code))
	if err != nil {
		return nil, err
	}
	if i == nil || !time.Now().UTC().Before(i.ExpiresAt) {
		return nil, domainerrors.ErrInvalidInvitation
	}
	if err := repo.LockPhone(ctx, i.Phone); err != nil {
		return nil, err
	}
	if err := f.JobSeekerRepository().LockAccount(ctx, i.JobSeekerID); err != nil {
		if errors.Is(err, domainerrors.ErrUnauthenticated) {
			return nil, domainerrors.ErrInvalidInvitation
		}
		return nil, err
	}
	i, err = repo.FindInvitation(ctx, invitationHash(code))
	if err != nil {
		return nil, err
	}
	if i == nil || !time.Now().UTC().Before(i.ExpiresAt) {
		return nil, domainerrors.ErrInvalidInvitation
	}
	seeker, err := f.JobSeekerRepository().FindByID(ctx, i.JobSeekerID)
	if err != nil {
		return nil, err
	}
	if seeker == nil || seeker.ParentID != nil || seeker.PhoneVerifiedAt.IsZero() {
		return nil, domainerrors.ErrInvalidInvitation
	}
	return i, nil
}

func (s *ParentService) ValidateInvitation(ctx context.Context, code string) (*usecase.ParentInvitationInfo, error) {
	var result *usecase.ParentInvitationInfo
	err := s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		i, err := lockedParentInvitation(ctx, f, code)
		if err != nil {
			return err
		}
		parent, err := f.ParentRepository().FindByPhone(ctx, i.Phone)
		if err != nil {
			return err
		}
		if parent != nil {
			return domainerrors.ErrSignupConflict
		}
		seeker, err := f.JobSeekerRepository().FindByID(ctx, i.JobSeekerID)
		if err != nil {
			return err
		}
		result = &usecase.ParentInvitationInfo{FirstName: i.FirstName, LastName: i.LastName, Phone: i.Phone, ExpiresAt: i.ExpiresAt, JobSeekerFirstName: seeker.FirstName}
		return nil
	})
	return result, err
}

func (s *ParentService) Signup(ctx context.Context, in usecase.SignupParent) error {
	if in.Relationship != "father" && in.Relationship != "mother" {
		return invalidParent("relationship must be father or mother")
	}
	if len(in.Password) < 8 || len(in.Password) > 72 {
		return invalidParent("password must contain 8 to 72 bytes")
	}
	if !in.ConsentAccepted {
		return invalidParent("parent consent is required")
	}
	if in.IdentityDocument.Body == nil || in.ConsentVideo.Body == nil {
		return domainerrors.ErrInvalidUpload
	}
	ext := strings.ToLower(filepath.Ext(in.IdentityDocument.Name))
	if ext != ".png" && ext != ".jpg" && ext != ".jpeg" {
		return fmt.Errorf("%w: parent identity document must be PNG or JPEG", domainerrors.ErrInvalidUpload)
	}
	var uploaded, replaced []string
	err := s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		i, err := lockedParentInvitation(ctx, f, in.Code)
		if err != nil {
			return err
		}
		repo := f.ParentRepository()
		existing, err := repo.FindByPhone(ctx, i.Phone)
		if err != nil {
			return err
		}
		if existing != nil {
			return domainerrors.ErrSignupConflict
		}
		previous, err := repo.GetPending(ctx, i.Phone)
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
		objects, err := uploadEmployerFiles(ctx, s.storage, s.media, "parents", []employerUpload{{kind: "identity", file: &in.IdentityDocument}, {kind: "consent-video", file: &in.ConsentVideo, video: true}}, &uploaded)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		registration := &entities.Parent{FirstName: i.FirstName, LastName: i.LastName, Phone: i.Phone, Relationship: in.Relationship, PasswordHash: hash, IdentityDocument: objects["identity"], ConsentVideo: objects["consent-video"], ConsentAcceptedAt: now, CreatedAt: now}
		pending := &entities.PendingParentSignup{InvitationHash: i.CodeHash, Relationship: in.Relationship, Registration: registration}
		if err := s.sendParentChallenge(ctx, repo, pending); err != nil {
			return err
		}
		if previous != nil {
			replaced = parentObjectKeys(previous.Registration)
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

func parentObjectKeys(p *entities.Parent) []string {
	var keys []string
	if p != nil {
		if p.IdentityDocument != nil {
			keys = append(keys, p.IdentityDocument.Key)
		}
		if p.ConsentVideo != nil {
			keys = append(keys, p.ConsentVideo.Key)
		}
	}
	return keys
}
func (s *ParentService) sendParentChallenge(ctx context.Context, repo ports.ParentRepository, p *entities.PendingParentSignup) error {
	challenge, code, err := newSignupChallenge(p.Registration.Phone, time.Now().UTC())
	if err != nil {
		return err
	}
	p.Challenge = challenge
	if err := repo.SavePending(ctx, p); err != nil {
		return err
	}
	if err := s.sender.SendOTP(ctx, "98"+p.Registration.Phone[1:], code); err != nil {
		return domainerrors.ErrSMSUnavailable
	}
	return nil
}
func validParentPending(p *entities.PendingParentSignup, i *entities.ParentInvitation) bool {
	return p != nil && p.Challenge != nil && p.Registration != nil && p.InvitationHash == i.CodeHash && p.Registration.Phone == i.Phone && time.Now().UTC().Before(p.Registration.CreatedAt.Add(pendingSignupLifetime))
}
func (s *ParentService) ResendOTP(ctx context.Context, code string) error {
	return s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		i, err := lockedParentInvitation(ctx, f, code)
		if err != nil {
			return err
		}
		repo := f.ParentRepository()
		p, err := repo.GetPending(ctx, i.Phone)
		if err != nil {
			return err
		}
		if !validParentPending(p, i) {
			return invalidParent("submit parent signup before requesting an OTP")
		}
		if time.Since(p.Challenge.SentAt) < otpResendDelay {
			return domainerrors.ErrOTPTooSoon
		}
		return s.sendParentChallenge(ctx, repo, p)
	})
}
func (s *ParentService) VerifyOTP(ctx context.Context, code, otpCode string) (*entities.Parent, error) {
	var result *entities.Parent
	var verifyErr error
	err := s.uow.WithTransaction(func(f ports.RepositoryFactory) error {
		i, err := lockedParentInvitation(ctx, f, code)
		if err != nil {
			return err
		}
		repo := f.ParentRepository()
		p, err := repo.GetPending(ctx, i.Phone)
		if err != nil {
			return err
		}
		if !validParentPending(p, i) || !time.Now().UTC().Before(p.Challenge.ExpiresAt) || p.Challenge.Attempts >= maxOTPAttempts {
			verifyErr = domainerrors.ErrInvalidOTP
			return nil
		}
		actual := otpHash(p.Challenge.Salt, otpCode)
		if !codePattern.MatchString(otpCode) || subtle.ConstantTimeCompare([]byte(actual), []byte(p.Challenge.CodeHash)) != 1 {
			p.Challenge.Attempts++
			verifyErr = domainerrors.ErrInvalidOTP
			return repo.SavePending(ctx, p)
		}
		existing, err := repo.FindByPhone(ctx, i.Phone)
		if err != nil {
			return err
		}
		if existing != nil {
			return domainerrors.ErrSignupConflict
		}
		result = p.Registration
		result.PhoneVerifiedAt = time.Now().UTC()
		if err := repo.Create(ctx, result); err != nil {
			return err
		}
		if err := repo.SetSignupStage(ctx, i.JobSeekerID, entities.SignupSecondStageCompleted, &result.ID, p.Relationship); err != nil {
			return err
		}
		if err := repo.DeleteInvitation(ctx, i.JobSeekerID); err != nil {
			return err
		}
		return repo.DeletePending(ctx, i.Phone)
	})
	if err != nil {
		return nil, err
	}
	if verifyErr != nil {
		return nil, verifyErr
	}
	return result, nil
}
