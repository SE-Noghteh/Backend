package service

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type employerMemory struct {
	signupMemory
	employer   *entities.IndividualEmployer
	pending    *entities.PendingIndividualEmployerSignup
	pendingErr error
	objects    map[string]valueobjects.StoredObject
	putCount   int
	failPut    int
	deleted    []string
	mediaErr   error
}

func (m *employerMemory) Factory() ports.RepositoryFactory                         { return m }
func (m *employerMemory) EmployerRepository() ports.EmployerRepository             { return m }
func (m *employerMemory) EmployerSignupRepository() ports.EmployerSignupRepository { return m }
func cloneEmployerPending(in *entities.PendingIndividualEmployerSignup) *entities.PendingIndividualEmployerSignup {
	if in == nil {
		return nil
	}
	registration := *in.Registration
	return &entities.PendingIndividualEmployerSignup{Challenge: cloneOTP(in.Challenge), Registration: &registration}
}
func (m *employerMemory) GetSignup(_ context.Context, phone string) (*entities.PendingIndividualEmployerSignup, error) {
	if m.pending == nil || m.pending.Challenge.Phone != phone {
		return nil, nil
	}
	return cloneEmployerPending(m.pending), nil
}
func (m *employerMemory) SaveSignup(_ context.Context, pending *entities.PendingIndividualEmployerSignup) error {
	if m.pendingErr != nil {
		return m.pendingErr
	}
	m.pending = cloneEmployerPending(pending)
	return nil
}
func (m *employerMemory) DeleteSignup(context.Context, string) error { m.pending = nil; return nil }

func (m *employerMemory) EmployerLoginAttemptRepository() ports.LoginAttemptRepository {
	return &m.signupMemory
}
func (m *employerMemory) WithTransaction(fn func(ports.RepositoryFactory) error) error {
	pending, attempt, employer := cloneEmployerPending(m.pending), cloneAttempt(m.attempt), m.employer
	err := fn(m)
	if err != nil {
		m.pending, m.attempt, m.employer = pending, attempt, employer
	}
	return err
}
func (m *employerMemory) IndividualExists(_ context.Context, phone, national string) (bool, error) {
	return m.employer != nil && (m.employer.Phone == phone || m.employer.NationalCode == national), nil
}
func (m *employerMemory) CreateIndividual(_ context.Context, e *entities.IndividualEmployer) error {
	if m.createErr != nil {
		return m.createErr
	}
	e.Employer = entities.Employer{ID: 1, Type: enums.EmployerIndividual}
	m.employer = e
	return nil
}
func (m *employerMemory) FindIndividualByPhone(_ context.Context, phone string) (*entities.IndividualEmployer, error) {
	if m.employer != nil && m.employer.Phone == phone {
		return m.employer, nil
	}
	return nil, nil
}
func (m *employerMemory) FindIndividualByID(_ context.Context, id uint64) (*entities.IndividualEmployer, error) {
	if m.employer != nil && m.employer.Employer.ID == id {
		return m.employer, nil
	}
	return nil, nil
}
func (m *employerMemory) Put(_ context.Context, key string, body io.ReadSeeker, contentType string, size int64) (*valueobjects.StoredObject, error) {
	m.putCount++
	if m.putCount == m.failPut {
		return nil, errors.New("storage down")
	}
	object := valueobjects.StoredObject{Key: key, ContentType: contentType, Size: size}
	if m.objects == nil {
		m.objects = map[string]valueobjects.StoredObject{}
	}
	m.objects[key] = object
	return &object, nil
}
func (m *employerMemory) Delete(_ context.Context, key string) error {
	m.deleted = append(m.deleted, key)
	delete(m.objects, key)
	return nil
}
func (m *employerMemory) PresignGet(_ context.Context, key string, _ time.Duration) (string, error) {
	if _, ok := m.objects[key]; !ok {
		return "", errors.New("missing object")
	}
	return "https://private.example/signed", nil
}
func (m *employerMemory) Document(_ context.Context, file ports.UploadFile) (*ports.ValidatedFile, error) {
	if m.mediaErr != nil {
		return nil, m.mediaErr
	}
	if file.Body == nil {
		return nil, domainerrors.ErrInvalidUpload
	}
	return &ports.ValidatedFile{UploadFile: file, ContentType: "application/pdf", Size: 10, Extension: ".pdf"}, nil
}
func (m *employerMemory) Video(ctx context.Context, file ports.UploadFile) (*ports.ValidatedFile, error) {
	v, err := m.Document(ctx, file)
	if err == nil {
		v.Extension = ".mp4"
		v.ContentType = "video/mp4"
	}
	return v, err
}

type employerTokens struct{}

func (employerTokens) IssueAccessToken(_ uint64, role enums.AccountRole) (*valueobjects.AccessToken, error) {
	return &valueobjects.AccessToken{Value: string(role), ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (employerTokens) VerifyAccessToken(token string, role enums.AccountRole) (*valueobjects.Principal, error) {
	if token != string(role) {
		return nil, domainerrors.ErrUnauthenticated
	}
	return &valueobjects.Principal{ID: 1, TokenID: "employer-test-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func employerFixture() usecase.SignupIndividualEmployer {
	return usecase.SignupIndividualEmployer{FirstName: "علی", LastName: "رضایی", FatherName: "حسن", NationalCode: "0013547895", Gender: enums.GenderMale,
		BirthDate: valueobjects.JalaliDate{Year: 1370, Month: 1, Day: 1}, Province: enums.ProvinceTehran, City: enums.CityTehran, Address: "خیابان آزادی", PostalCode: "1234567890",
		Phone: "09123456789", Password: "EmployerTest123!",
		IdentityDocument: ports.UploadFile{Name: "id.pdf", Body: strings.NewReader("%PDF-1.7\n")}, VerificationVideo: ports.UploadFile{Name: "video.mp4", Body: strings.NewReader("video")}}
}
func employerService(m *employerMemory) *IndividualEmployerService {
	return NewIndividualEmployerService(m, m, testPasswords{}, employerTokens{}, m, m)
}

func submitEmployer(t *testing.T, s *IndividualEmployerService, m *employerMemory) usecase.SignupIndividualEmployer {
	t.Helper()
	in := employerFixture()
	if err := s.Signup(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	return in
}

func verifyEmployer(t *testing.T, s *IndividualEmployerService, m *employerMemory) *entities.IndividualEmployer {
	t.Helper()
	e, err := s.VerifyOTP(context.Background(), usecase.VerifySignupOTP{Phone: m.pending.Challenge.Phone, OTP: m.smsCode})
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestIndividualEmployerSignupSendsOTPBeforeAccountCreation(t *testing.T) {
	for _, license := range []bool{false, true} {
		t.Run(map[bool]string{false: "nullable license", true: "with license"}[license], func(t *testing.T) {
			m := &employerMemory{}
			s := employerService(m)
			in := employerFixture()
			ctx := context.Background()
			if license {
				in.BusinessLicense = &ports.UploadFile{Name: "license.pdf", Body: strings.NewReader("%PDF")}
			}
			if err := s.Signup(ctx, in); err != nil {
				t.Fatal(err)
			}
			if m.smsCount != 1 || m.employer != nil || m.pending == nil {
				t.Fatal("signup must automatically send OTP and persist only pending information")
			}
			if m.pending.Registration.PasswordHash == in.Password || !m.pending.Registration.PhoneVerifiedAt.IsZero() {
				t.Fatal("pending signup contains a raw password or verified phone")
			}
			if _, err := s.Login(ctx, in.Phone, in.Password); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
				t.Fatal("unverified employer logged in")
			}
			expected := 2
			if license {
				expected = 3
			}
			if len(m.objects) != expected {
				t.Fatal("documents were not stored once")
			}
			e := verifyEmployer(t, s, m)
			if e.Employer.ID != 1 || e.Employer.Type != enums.EmployerIndividual || e.ReviewStatus != enums.EmployerPendingReview || e.PhoneVerifiedAt.IsZero() || m.pending != nil {
				t.Fatal("verification did not create a verified account and consume pending signup")
			}
			if (e.BusinessLicense != nil) != license {
				t.Fatal("license nullability changed")
			}
			if m.putCount != expected {
				t.Fatal("verification re-uploaded documents")
			}
			if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatal("OTP replay accepted")
			}
			login, err := s.Login(ctx, "+989123456789", in.Password)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Authenticate(ctx, login.AccessToken.Value); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Authenticate(ctx, string(enums.RoleJobSeeker)); !errors.Is(err, domainerrors.ErrUnauthenticated) {
				t.Fatal("job seeker token accepted")
			}
			if _, err := s.DocumentURL(ctx, login.AccessToken.Value, "identity"); err != nil {
				t.Fatal(err)
			}
			if err := s.Logout(ctx, login.AccessToken.Value); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Authenticate(ctx, login.AccessToken.Value); !errors.Is(err, domainerrors.ErrUnauthenticated) {
				t.Fatal("logged-out token accepted")
			}
			if err := s.Signup(ctx, in); !errors.Is(err, domainerrors.ErrSignupConflict) {
				t.Fatal("duplicate accepted")
			}
		})
	}
}

func TestEmployerOTPExpiryLockoutAndCooldown(t *testing.T) {
	for _, scenario := range []string{"wrong", "malformed", "expired code", "expired submission"} {
		t.Run(scenario, func(t *testing.T) {
			m := &employerMemory{}
			s := employerService(m)
			in := submitEmployer(t, s, m)
			ctx := context.Background()
			if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, domainerrors.ErrOTPTooSoon) {
				t.Fatal(err)
			}
			wrong := "000000"
			if wrong == m.smsCode {
				wrong = "111111"
			}
			switch scenario {
			case "malformed":
				wrong = "abc"
			case "expired code":
				m.pending.Challenge.ExpiresAt = time.Now().Add(-time.Second)
			case "expired submission":
				m.pending.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
			}
			for i := 0; i < 5; i++ {
				if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: wrong}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
					t.Fatal(err)
				}
			}
			if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatal("expired/locked code accepted")
			}
			if m.putCount != 2 || m.employer != nil {
				t.Fatal("verification uploaded documents or created an account")
			}
		})
	}
}

func TestEmployerResendRequiresSubmissionAndDoesNotReupload(t *testing.T) {
	m := &employerMemory{}
	s := employerService(m)
	ctx := context.Background()
	in := employerFixture()
	if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, domainerrors.ErrNoPendingSignup) {
		t.Fatal("OTP sent before receiving employer information")
	}
	submitEmployer(t, s, m)
	m.pending.Challenge.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.RequestOTP(ctx, in.Phone); err != nil {
		t.Fatal(err)
	}
	if m.putCount != 2 || m.smsCount != 2 {
		t.Fatal("resend uploaded files or did not send SMS")
	}
	m.pending.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
	if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, domainerrors.ErrNoPendingSignup) {
		t.Fatal("expired submission allowed resend")
	}
}

func TestEmployerSignupFailureCleansNewObjects(t *testing.T) {
	for _, scenario := range []string{"storage", "database", "media", "sms"} {
		t.Run(scenario, func(t *testing.T) {
			m := &employerMemory{}
			s := employerService(m)
			switch scenario {
			case "storage":
				m.failPut = 2
			case "database":
				m.pendingErr = errors.New("db down")
			case "media":
				m.mediaErr = domainerrors.ErrInvalidUpload
			case "sms":
				m.smsErr = errors.New("provider down")
			}
			if err := s.Signup(context.Background(), employerFixture()); err == nil {
				t.Fatal("failure ignored")
			}
			if len(m.objects) != 0 || m.employer != nil || m.pending != nil {
				t.Fatal("failed signup left files or database state")
			}
		})
	}
}

func TestEmployerVerificationFailureRetainsSubmissionAndFiles(t *testing.T) {
	m := &employerMemory{}
	s := employerService(m)
	in := submitEmployer(t, s, m)
	m.createErr = errors.New("db down")
	if _, err := s.VerifyOTP(context.Background(), usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}); err == nil {
		t.Fatal("database failure ignored")
	}
	if m.employer != nil || m.pending == nil || len(m.objects) != 2 || !m.pending.Registration.PhoneVerifiedAt.IsZero() {
		t.Fatal("failed verification changed pending information or files")
	}
	m.createErr = nil
	verifyEmployer(t, s, m)
}

func TestEmployerReplacementCleansOldFilesOnlyAfterSuccess(t *testing.T) {
	m := &employerMemory{}
	s := employerService(m)
	submitEmployer(t, s, m)
	oldKeys := employerObjectKeys(m.pending.Registration)
	m.pending.Challenge.SentAt = time.Now().Add(-2 * time.Minute)
	updated := employerFixture()
	updated.FirstName = "Updated"
	if err := s.Signup(context.Background(), updated); err != nil {
		t.Fatal(err)
	}
	if len(m.objects) != 2 || m.pending.Registration.FirstName != "Updated" {
		t.Fatal("replacement not persisted")
	}
	for _, key := range oldKeys {
		if _, exists := m.objects[key]; exists {
			t.Fatal("old upload retained")
		}
	}
	m.pending.Challenge.SentAt = time.Now().Add(-2 * time.Minute)
	m.smsErr = errors.New("provider down")
	keys := employerObjectKeys(m.pending.Registration)
	if err := s.Signup(context.Background(), employerFixture()); !errors.Is(err, domainerrors.ErrSMSUnavailable) {
		t.Fatal(err)
	}
	for _, key := range keys {
		if _, exists := m.objects[key]; !exists {
			t.Fatal("failed replacement deleted existing submission documents")
		}
	}
	if m.pending.Registration.FirstName != "Updated" {
		t.Fatal("failed replacement changed pending information")
	}
}

func TestEmployerValidationBeforeUploadsOrSMS(t *testing.T) {
	for _, mutate := range []func(*usecase.SignupIndividualEmployer){
		func(in *usecase.SignupIndividualEmployer) { in.FatherName = "" }, func(in *usecase.SignupIndividualEmployer) { in.Password = "weak-password" },

		func(in *usecase.SignupIndividualEmployer) { in.Province = enums.ProvinceFars }, func(in *usecase.SignupIndividualEmployer) { in.PostalCode = "123" },
		func(in *usecase.SignupIndividualEmployer) { in.Email = "bad-email" }, func(in *usecase.SignupIndividualEmployer) { in.Website = "javascript:bad" },
	} {
		m := &employerMemory{}
		s := employerService(m)
		in := employerFixture()
		mutate(&in)
		if err := s.Signup(context.Background(), in); !errors.Is(err, domainerrors.ErrInvalidSignup) {
			t.Fatal("invalid input accepted")
		}
		if m.putCount != 0 || m.smsCount != 0 {
			t.Fatal("invalid signup uploaded files or sent SMS")
		}
	}
}

func TestEmployerLoginThrottle(t *testing.T) {
	m := &employerMemory{}
	s := employerService(m)
	in := submitEmployer(t, s, m)
	verifyEmployer(t, s, m)
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := s.Login(ctx, in.Phone, "WrongPassword123!"); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
			t.Fatal(err)
		}
	}
	if _, err := s.Login(ctx, in.Phone, in.Password); !errors.Is(err, domainerrors.ErrLoginThrottled) {
		t.Fatal("lockout bypassed")
	}
	m.attempt.WindowStartedAt = time.Now().Add(-16 * time.Minute)
	if _, err := s.Login(ctx, in.Phone, in.Password); err != nil {
		t.Fatal(err)
	}
}

func (m *employerMemory) LegalExists(context.Context, string, string) (bool, error) {
	return false, nil
}
func (m *employerMemory) CreateLegal(context.Context, *entities.LegalEmployer) error { return nil }
func (m *employerMemory) FindLegalByPhone(context.Context, string) (*entities.LegalEmployer, error) {
	return nil, nil
}
func (m *employerMemory) FindLegalByID(context.Context, uint64) (*entities.LegalEmployer, error) {
	return nil, nil
}
