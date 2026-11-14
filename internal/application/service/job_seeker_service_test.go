package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	domainpostgres "zedlancer/backend/internal/domain/ports/postgres"
	"zedlancer/backend/internal/domain/valueobjects"
)

type signupMemory struct {
	phoneChange    *entities.SignupOTP
	occupiedPhone  string
	otp            *entities.SignupOTP
	seeker         *entities.JobSeeker
	attempt        *entities.LoginAttempt
	smsCode        string
	smsDestination string
	smsCount       int
	smsErr         error
	createErr      error
	saveErr        error
	commitErr      error
	revoked        map[string]time.Time
	revocationErr  error
}

func (m *signupMemory) Factory() ports.RepositoryFactory         { return m }
func (m *signupMemory) ParentRepository() ports.ParentRepository { return nil }
func (m *signupMemory) WithTransaction(fn func(ports.RepositoryFactory) error) error {
	previousOTP, previousSeeker, previousAttempt := cloneOTP(m.otp), cloneSeeker(m.seeker), cloneAttempt(m.attempt)
	previousPhoneChange := cloneOTP(m.phoneChange)
	err := fn(m)
	if err == nil {
		err = m.commitErr
	}
	if err != nil {
		m.otp, m.seeker, m.attempt = previousOTP, previousSeeker, previousAttempt
		m.phoneChange = previousPhoneChange
	}
	return err
}
func cloneSeeker(in *entities.JobSeeker) *entities.JobSeeker {
	if in == nil {
		return nil
	}
	out := *in
	out.Interests = append([]enums.Interest(nil), in.Interests...)
	return &out
}
func cloneOTP(in *entities.SignupOTP) *entities.SignupOTP {
	if in == nil {
		return nil
	}
	out := *in
	out.Registration = cloneSeeker(in.Registration)
	return &out
}
func cloneAttempt(in *entities.LoginAttempt) *entities.LoginAttempt {
	if in == nil {
		return nil
	}
	out := *in
	return &out
}
func (m *signupMemory) UserRepository() domainpostgres.UserRepository              { return nil }
func (m *signupMemory) JobSeekerRepository() ports.JobSeekerRepository             { return m }
func (m *signupMemory) TokenRevocationRepository() ports.TokenRevocationRepository { return m }
func (m *signupMemory) IsRevoked(_ context.Context, tokenID string) (bool, error) {
	_, revoked := m.revoked[tokenID]
	return revoked, m.revocationErr
}
func (m *signupMemory) Revoke(_ context.Context, tokenID string, expiresAt time.Time) error {
	if m.revocationErr != nil {
		return m.revocationErr
	}
	if m.revoked == nil {
		m.revoked = make(map[string]time.Time)
	}
	m.revoked[tokenID] = expiresAt
	return nil
}
func (m *signupMemory) LockPhone(context.Context, string) error { return nil }
func (m *signupMemory) Exists(_ context.Context, phone, national string) (bool, error) {
	return m.seeker != nil && (m.seeker.Phone == phone || m.seeker.NationalCode == national), nil
}
func (m *signupMemory) GetOTP(_ context.Context, phone string) (*entities.SignupOTP, error) {
	if m.otp == nil || m.otp.Phone != phone {
		return nil, nil
	}
	return cloneOTP(m.otp), nil
}
func (m *signupMemory) SaveOTP(_ context.Context, otp *entities.SignupOTP) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.otp = cloneOTP(otp)
	return nil
}

func TestJobSeekerVideoRequiredAndUploadFailures(t *testing.T) {
	for _, scenario := range []string{"missing", "invalid", "storage", "sms", "save", "commit"} {
		t.Run(scenario, func(t *testing.T) {
			m := &signupMemory{}
			uploads := &employerMemory{}
			s := NewJobSeekerService(m, m, testPasswords{}, uploads, uploads)
			in := validSignup()
			switch scenario {
			case "missing":
				in.VerificationVideo = ports.UploadFile{}
			case "invalid":
				uploads.mediaErr = domainerrors.ErrInvalidUpload
			case "storage":
				uploads.failPut = 1
			case "sms":
				m.smsErr = errors.New("SMS unavailable")
			case "save":
				m.saveErr = errors.New("save failed")
			case "commit":
				m.commitErr = errors.New("commit failed")
			}
			if err := s.Signup(context.Background(), in); err == nil {
				t.Fatal("failed upload/signup accepted")
			}
			if m.otp != nil || m.seeker != nil || len(uploads.objects) != 0 {
				t.Fatal("failed signup left pending data or uploaded objects")
			}
			if scenario == "missing" || scenario == "invalid" {
				if uploads.putCount != 0 || m.smsCount != 0 {
					t.Fatal("invalid video uploaded or sent OTP")
				}
			}
		})
	}
}

func TestJobSeekerVideoLifecycle(t *testing.T) {
	m := &signupMemory{}
	uploads := &employerMemory{}
	s := NewJobSeekerService(m, m, testPasswords{}, uploads, uploads)
	ctx := context.Background()
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	oldKey := m.otp.Registration.VerificationVideo.Key
	if !strings.HasPrefix(oldKey, "job-seekers/") || strings.Contains(oldKey, validSignup().Phone) {
		t.Fatal("video key must be private and random")
	}
	m.otp.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.RequestOTP(ctx, validSignup().Phone); err != nil {
		t.Fatal(err)
	}
	if uploads.putCount != 1 || m.otp.Registration.VerificationVideo.Key != oldKey {
		t.Fatal("resending OTP changed the video")
	}
	m.otp.SentAt = time.Now().Add(-2 * time.Minute)
	m.smsErr = errors.New("SMS failed")
	if err := s.Signup(ctx, validSignup()); err == nil {
		t.Fatal("failed replacement accepted")
	}
	if m.otp.Registration.VerificationVideo.Key != oldKey || len(uploads.objects) != 1 {
		t.Fatal("failed replacement removed the previous video")
	}
	if _, ok := uploads.objects[oldKey]; !ok {
		t.Fatal("previous video was deleted")
	}
	m.smsErr = nil
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	newKey := m.otp.Registration.VerificationVideo.Key
	if oldKey == newKey || len(uploads.objects) != 1 {
		t.Fatal("replacement must keep only the new video")
	}
	if _, ok := uploads.objects[oldKey]; ok {
		t.Fatal("replacement left the old video")
	}
	seeker, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode})
	if err != nil {
		t.Fatal(err)
	}
	if seeker.VerificationVideo == nil || seeker.VerificationVideo.Key != newKey {
		t.Fatal("verification lost the video reference")
	}
	if _, err := s.VideoURL(ctx, seeker.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.VideoURL(ctx, seeker.ID+1); !errors.Is(err, domainerrors.ErrUnauthenticated) {
		t.Fatal("missing account received a download URL")
	}
}

func TestJobSeekerLegacyPendingRequiresVideo(t *testing.T) {
	m := &signupMemory{}
	s := newJobSeekerService(m)
	ctx := context.Background()
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	m.otp.Registration.VerificationVideo = nil
	m.otp.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.RequestOTP(ctx, validSignup().Phone); !errors.Is(err, domainerrors.ErrNoPendingSignup) {
		t.Fatal("legacy pending signup without video was resent")
	}
	if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("legacy pending signup without video created an account")
	}
}
func (m *signupMemory) DeleteOTP(context.Context, string) error { m.otp = nil; return nil }
func (m *signupMemory) Create(_ context.Context, seeker *entities.JobSeeker) error {
	if m.createErr != nil {
		return m.createErr
	}
	seeker.ID = 1
	m.seeker = cloneSeeker(seeker)
	return nil
}
func (m *signupMemory) FindByPhone(_ context.Context, phone string) (*entities.JobSeeker, error) {
	if m.occupiedPhone != "" && m.occupiedPhone == phone {
		return &entities.JobSeeker{ID: 2, Phone: phone}, nil
	}
	if m.seeker != nil && m.seeker.Phone == phone {
		return cloneSeeker(m.seeker), nil
	}
	return nil, nil
}
func (m *signupMemory) FindByID(_ context.Context, id uint64) (*entities.JobSeeker, error) {
	if m.seeker != nil && m.seeker.ID == id {
		return cloneSeeker(m.seeker), nil
	}
	return nil, nil
}
func (m *signupMemory) GetLoginAttempt(_ context.Context, phone string) (*entities.LoginAttempt, error) {
	if m.attempt != nil && m.attempt.Phone == phone {
		return cloneAttempt(m.attempt), nil
	}
	return nil, nil
}
func (m *signupMemory) SaveLoginAttempt(_ context.Context, in *entities.LoginAttempt) error {
	m.attempt = cloneAttempt(in)
	return nil
}
func (m *signupMemory) DeleteLoginAttempt(context.Context, string) error { m.attempt = nil; return nil }
func (m *signupMemory) SendOTP(_ context.Context, destination, code string) error {
	expected := m.smsDestination
	if expected == "" {
		expected = "989123456789"
	}
	if destination != expected {
		return errors.New("wrong destination")
	}
	if m.smsErr != nil {
		return m.smsErr
	}
	m.smsCode = code
	m.smsCount++
	return nil
}

type testPasswords struct{}

func (testPasswords) Hash(password string) (string, error) { return "hash:" + password, nil }
func (testPasswords) Matches(hash, password string) bool {
	return hash != "" && hash == "hash:"+password
}

func newJobSeekerService(m *signupMemory) *JobSeekerService {
	uploads := &employerMemory{}
	return NewJobSeekerService(m, m, testPasswords{}, uploads, uploads)
}

func validSignup() usecase.SignupJobSeeker {
	return usecase.SignupJobSeeker{FirstName: "علی", LastName: "رضایی", NationalCode: "0013547895", City: enums.CityTehran, Address: "خیابان آزادی", BirthDate: valueobjects.JalaliDate{Year: 1388, Month: 1, Day: 1}, Gender: enums.GenderMale, Interests: []enums.Interest{enums.InterestProgramming}, Phone: "09123456789", Password: "a-good-password", VerificationVideo: ports.UploadFile{Name: "video.mp4", Body: strings.NewReader("video")}}
}

func TestSignupAutomaticallySendsOTPAndRequiresVerification(t *testing.T) {
	m := &signupMemory{}
	s := newJobSeekerService(m)
	ctx := context.Background()
	in := validSignup()
	in.Phone = "+989123456789"
	if err := s.Signup(ctx, in); err != nil {
		t.Fatal(err)
	}
	if m.smsCount != 1 || m.seeker != nil || m.otp.Registration == nil {
		t.Fatal("signup must send OTP and persist only a pending registration")
	}
	if m.otp.CodeHash == m.smsCode || m.otp.Salt == "" {
		t.Fatal("OTP was not hashed")
	}
	if m.otp.Registration.PasswordHash == in.Password || !strings.HasPrefix(m.otp.Registration.PasswordHash, "hash:") {
		t.Fatal("plaintext password persisted")
	}
	if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, domainerrors.ErrOTPTooSoon) {
		t.Fatalf("cooldown: %v", err)
	}
	verify := usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}
	seeker, err := s.VerifyOTP(ctx, verify)
	if err != nil {
		t.Fatal(err)
	}
	if m.otp != nil || seeker.ID == 0 || seeker.PhoneVerifiedAt.IsZero() || seeker.Phone != "09123456789" {
		t.Fatal("OTP must be consumed and verified account persisted")
	}
	if _, err := s.VerifyOTP(ctx, verify); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatalf("OTP replay: %v", err)
	}
	if err := s.Signup(ctx, in); !errors.Is(err, domainerrors.ErrSignupConflict) {
		t.Fatalf("duplicate signup: %v", err)
	}
}

func TestOTPExpiryAndLockout(t *testing.T) {
	for _, scenario := range []string{"expired code", "expired registration", "wrong code", "malformed code"} {
		t.Run(scenario, func(t *testing.T) {
			m := &signupMemory{}
			s := newJobSeekerService(m)
			ctx := context.Background()
			if err := s.Signup(ctx, validSignup()); err != nil {
				t.Fatal(err)
			}
			wrong := "000000"
			if wrong == m.smsCode {
				wrong = "111111"
			}
			if scenario == "expired code" {
				m.otp.ExpiresAt = time.Now().Add(-time.Second)
			}
			if scenario == "expired registration" {
				m.otp.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
			}
			if scenario == "malformed code" {
				wrong = "abc"
			}
			for i := 0; i < maxOTPAttempts; i++ {
				if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: wrong}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
					t.Fatalf("attempt %d: %v", i, err)
				}
			}
			if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatalf("expired/locked code accepted: %v", err)
			}
			if strings.HasSuffix(scenario, "code") && scenario != "expired code" && m.otp.Attempts != 5 {
				t.Fatal("failed attempts not committed")
			}
		})
	}
}

func TestResendRequiresPendingProfileAndInvalidatesOldCode(t *testing.T) {
	m := &signupMemory{}
	s := newJobSeekerService(m)
	ctx := context.Background()
	if err := s.RequestOTP(ctx, validSignup().Phone); !errors.Is(err, domainerrors.ErrNoPendingSignup) {
		t.Fatal(err)
	}
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	oldSalt := m.otp.Salt
	m.otp.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.RequestOTP(ctx, validSignup().Phone); err != nil {
		t.Fatal(err)
	}
	if m.otp.Salt == oldSalt || m.smsCount != 2 {
		t.Fatal("resend must replace the challenge")
	}
	m.otp.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
	if err := s.RequestOTP(ctx, validSignup().Phone); !errors.Is(err, domainerrors.ErrNoPendingSignup) {
		t.Fatal(err)
	}
}

func TestSignupRollsBackOnSMSFailure(t *testing.T) {
	m := &signupMemory{smsErr: errors.New("provider down")}
	s := newJobSeekerService(m)
	if err := s.Signup(context.Background(), validSignup()); !errors.Is(err, domainerrors.ErrSMSUnavailable) {
		t.Fatal(err)
	}
	if m.otp != nil || m.seeker != nil {
		t.Fatal("failed delivery must roll back pending signup")
	}
}

func TestVerificationRetainsCodeOnDatabaseFailure(t *testing.T) {
	m := &signupMemory{}
	s := newJobSeekerService(m)
	ctx := context.Background()
	if err := s.Signup(ctx, validSignup()); err != nil {
		t.Fatal(err)
	}
	m.createErr = errors.New("database unavailable")
	if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: validSignup().Phone, OTP: m.smsCode}); err == nil {
		t.Fatal("database error not returned")
	}
	if m.otp == nil || m.seeker != nil {
		t.Fatal("failed creation must not consume code")
	}
}

func TestSignupRejectsInvalidProfileBeforeSendingSMS(t *testing.T) {
	for _, mutate := range []func(*usecase.SignupJobSeeker){
		func(in *usecase.SignupJobSeeker) { in.City = "invalid" },
		func(in *usecase.SignupJobSeeker) { in.Interests = []enums.Interest{"invalid"} },
		func(in *usecase.SignupJobSeeker) {
			in.Interests = []enums.Interest{enums.InterestDesign, enums.InterestDesign}
		},
		func(in *usecase.SignupJobSeeker) { in.NationalCode = "1111111111" },
		func(in *usecase.SignupJobSeeker) {
			in.BirthDate = valueobjects.JalaliDate{Year: 1404, Month: 12, Day: 30}
		},
		func(in *usecase.SignupJobSeeker) {
			in.BirthDate = valueobjects.JalaliDate{Year: 1500, Month: 1, Day: 1}
		},
		func(in *usecase.SignupJobSeeker) { in.Password = "short" },
		func(in *usecase.SignupJobSeeker) { in.Phone = "123" },
	} {
		m := &signupMemory{}
		s := newJobSeekerService(m)
		in := validSignup()
		mutate(&in)
		if err := s.Signup(context.Background(), in); !errors.Is(err, domainerrors.ErrInvalidSignup) {
			t.Fatalf("invalid input accepted: %v", err)
		}
		if m.smsCount != 0 || m.otp != nil {
			t.Fatal("invalid signup sent SMS or persisted data")
		}
	}
}

func (m *signupMemory) SignupOTPRepository() ports.SignupOTPRepository       { return m }
func (m *signupMemory) LoginAttemptRepository() ports.LoginAttemptRepository { return m }

func (m *signupMemory) EmployerRepository() ports.EmployerRepository                 { return nil }
func (m *signupMemory) EmployerSignupRepository() ports.EmployerSignupRepository     { return nil }
func (m *signupMemory) EmployerLoginAttemptRepository() ports.LoginAttemptRepository { return nil }

func (m *signupMemory) LegalEmployerSignupRepository() ports.LegalEmployerSignupRepository {
	return nil
}
