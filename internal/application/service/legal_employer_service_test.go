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
)

type legalEmployerMemory struct {
	employerMemory
	legal        *entities.LegalEmployer
	legalPending *entities.PendingLegalEmployerSignup
}

func cloneLegalPending(in *entities.PendingLegalEmployerSignup) *entities.PendingLegalEmployerSignup {
	if in == nil {
		return nil
	}
	e := *in.Registration
	e.Activities = append([]enums.Interest(nil), in.Registration.Activities...)
	return &entities.PendingLegalEmployerSignup{Challenge: cloneOTP(in.Challenge), Registration: &e}
}
func (m *legalEmployerMemory) Factory() ports.RepositoryFactory             { return m }
func (m *legalEmployerMemory) EmployerRepository() ports.EmployerRepository { return m }
func (m *legalEmployerMemory) LegalEmployerSignupRepository() ports.LegalEmployerSignupRepository {
	return m
}
func (m *legalEmployerMemory) GetSignup(_ context.Context, phone string) (*entities.PendingLegalEmployerSignup, error) {
	if m.legalPending == nil || m.legalPending.Challenge.Phone != phone {
		return nil, nil
	}
	return cloneLegalPending(m.legalPending), nil
}
func (m *legalEmployerMemory) SaveSignup(_ context.Context, p *entities.PendingLegalEmployerSignup) error {
	if m.pendingErr != nil {
		return m.pendingErr
	}
	m.legalPending = cloneLegalPending(p)
	return nil
}
func (m *legalEmployerMemory) DeleteSignup(context.Context, string) error {
	m.legalPending = nil
	return nil
}
func (m *legalEmployerMemory) WithTransaction(fn func(ports.RepositoryFactory) error) error {
	pending, attempt, employer := cloneLegalPending(m.legalPending), cloneAttempt(m.attempt), m.legal
	err := fn(m)
	if err != nil {
		m.legalPending, m.attempt, m.legal = pending, attempt, employer
	}
	return err
}
func (m *legalEmployerMemory) LegalExists(_ context.Context, phone, national string) (bool, error) {
	return m.legal != nil && (m.legal.Phone == phone || m.legal.CompanyNationalID == national), nil
}
func (m *legalEmployerMemory) CreateLegal(_ context.Context, e *entities.LegalEmployer) error {
	if m.createErr != nil {
		return m.createErr
	}
	e.Employer = entities.Employer{ID: 1, Type: enums.EmployerLegal}
	m.legal = e
	return nil
}
func (m *legalEmployerMemory) FindLegalByPhone(_ context.Context, phone string) (*entities.LegalEmployer, error) {
	if m.legal != nil && m.legal.Phone == phone {
		return m.legal, nil
	}
	return nil, nil
}
func (m *legalEmployerMemory) FindLegalByID(_ context.Context, id uint64) (*entities.LegalEmployer, error) {
	if m.legal != nil && m.legal.Employer.ID == id {
		return m.legal, nil
	}
	return nil, nil
}

func legalEmployerFixture() usecase.SignupLegalEmployer {
	individual := employerFixture()
	return usecase.SignupLegalEmployer{
		CompanyName: "شرکت نمونه", LegalEntityType: enums.LegalLimitedLiability, CompanyNationalID: "14012345678", RegistrationNumber: "123456", EconomicCode: "123456789012",
		Province: individual.Province, City: individual.City, Address: individual.Address, PostalCode: individual.PostalCode,
		RepresentativeFirstName: individual.FirstName, RepresentativeLastName: individual.LastName, RepresentativePosition: enums.RepresentativeCEO, RepresentativeGender: individual.Gender, RepresentativeNationalCode: individual.NationalCode,
		Activities: []enums.Interest{enums.InterestProgramming, enums.InterestDesign}, Phone: individual.Phone, Password: individual.Password,
		EstablishmentDocument: ports.UploadFile{Name: "gazette.pdf", Body: strings.NewReader("%PDF")}, IdentityDocument: individual.IdentityDocument, VerificationVideo: individual.VerificationVideo,
	}
}
func legalEmployerService(m *legalEmployerMemory) *LegalEmployerService {
	return NewLegalEmployerService(m, m, testPasswords{}, employerTokens{}, m, m)
}

func TestLegalEmployerSignupVerificationAndAuthentication(t *testing.T) {
	for _, licensed := range []bool{false, true} {
		t.Run(map[bool]string{false: "optional documents null", true: "licensed attorney"}[licensed], func(t *testing.T) {
			m := &legalEmployerMemory{}
			s := legalEmployerService(m)
			in := legalEmployerFixture()
			ctx := context.Background()
			expectedObjects := 3
			if licensed {
				in.RequiresActivityLicense = true
				in.RepresentativePosition = enums.RepresentativeAttorney
				in.ActivityLicense = &ports.UploadFile{Name: "license.pdf", Body: strings.NewReader("%PDF")}
				in.AuthorizationDocument = &ports.UploadFile{Name: "authority.pdf", Body: strings.NewReader("%PDF")}
				expectedObjects = 5
			}
			if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, domainerrors.ErrNoPendingSignup) {
				t.Fatal("OTP sent without information")
			}
			if err := s.Signup(ctx, in); err != nil {
				t.Fatal(err)
			}
			if m.smsCount != 1 || m.legal != nil || m.legalPending == nil || len(m.objects) != expectedObjects {
				t.Fatal("signup must store a pending submission, upload files and automatically send OTP")
			}
			pending := m.legalPending
			if pending.Registration.PasswordHash == in.Password || pending.Challenge.CodeHash == m.smsCode || !pending.Registration.PhoneVerifiedAt.IsZero() {
				t.Fatal("raw credentials saved or pending phone marked verified")
			}
			if _, err := s.Login(ctx, in.Phone, in.Password); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
				t.Fatal("unverified account logged in")
			}
			if err := s.RequestOTP(ctx, in.Phone); !errors.Is(err, domainerrors.ErrOTPTooSoon) {
				t.Fatal("resend cooldown bypassed")
			}
			verified, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode})
			if err != nil {
				t.Fatal(err)
			}
			if verified.Employer.Type != enums.EmployerLegal || verified.PhoneVerifiedAt.IsZero() || verified.ReviewStatus != enums.EmployerPendingReview || m.legalPending != nil {
				t.Fatal("verification did not finalize legal profile")
			}
			if (verified.ActivityLicense != nil) != licensed || (verified.AuthorizationDocument != nil) != licensed || m.putCount != expectedObjects {
				t.Fatal("document references changed or files were uploaded twice")
			}
			if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatal("OTP replay accepted")
			}
			login, err := s.Login(ctx, "+989123456789", in.Password)
			if err != nil {
				t.Fatal(err)
			}
			if login.AccessToken.Value != string(enums.RoleLegalEmployer) {
				t.Fatal("wrong JWT role")
			}
			if _, err := s.Authenticate(ctx, login.AccessToken.Value); err != nil {
				t.Fatal(err)
			}
			for _, role := range []enums.AccountRole{enums.RoleJobSeeker, enums.RoleIndividualEmployer} {
				if _, err := s.Authenticate(ctx, string(role)); !errors.Is(err, domainerrors.ErrUnauthenticated) {
					t.Fatal("foreign role accepted")
				}
			}
			for _, kind := range []string{"establishment", "identity", "video"} {
				if _, err := s.DocumentURL(ctx, login.AccessToken.Value, kind); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.DocumentURL(ctx, login.AccessToken.Value, "unknown"); !errors.Is(err, domainerrors.ErrInvalidSignup) {
				t.Fatal("unknown document accepted")
			}
			if err := s.Logout(ctx, login.AccessToken.Value); err != nil {
				t.Fatal(err)
			}
			if _, err := s.Authenticate(ctx, login.AccessToken.Value); !errors.Is(err, domainerrors.ErrUnauthenticated) {
				t.Fatal("revoked token accepted")
			}
			if err := s.Signup(ctx, in); !errors.Is(err, domainerrors.ErrSignupConflict) {
				t.Fatal("duplicate accepted")
			}
		})
	}
}

func TestLegalEmployerValidationBeforeSideEffects(t *testing.T) {
	tests := map[string]func(*usecase.SignupLegalEmployer){
		"company name":            func(in *usecase.SignupLegalEmployer) { in.CompanyName = " " },
		"entity type":             func(in *usecase.SignupLegalEmployer) { in.LegalEntityType = "other" },
		"national id":             func(in *usecase.SignupLegalEmployer) { in.CompanyNationalID = "123" },
		"empty national id":       func(in *usecase.SignupLegalEmployer) { in.CompanyNationalID = "" },
		"repeated national id":    func(in *usecase.SignupLegalEmployer) { in.CompanyNationalID = "00000000000" },
		"registration":            func(in *usecase.SignupLegalEmployer) { in.RegistrationNumber = "abc" },
		"economic code":           func(in *usecase.SignupLegalEmployer) { in.EconomicCode = "abc" },
		"location":                func(in *usecase.SignupLegalEmployer) { in.City = enums.CityIsfahan },
		"postal code":             func(in *usecase.SignupLegalEmployer) { in.PostalCode = "abc" },
		"representative name":     func(in *usecase.SignupLegalEmployer) { in.RepresentativeFirstName = "" },
		"representative position": func(in *usecase.SignupLegalEmployer) { in.RepresentativePosition = "other" },
		"representative gender":   func(in *usecase.SignupLegalEmployer) { in.RepresentativeGender = "other" },
		"representative code":     func(in *usecase.SignupLegalEmployer) { in.RepresentativeNationalCode = "1234567890" },
		"no activities":           func(in *usecase.SignupLegalEmployer) { in.Activities = nil },
		"unknown activity":        func(in *usecase.SignupLegalEmployer) { in.Activities = []enums.Interest{"unknown"} },
		"duplicate activity": func(in *usecase.SignupLegalEmployer) {
			in.Activities = []enums.Interest{enums.InterestDesign, enums.InterestDesign}
		},
		"missing license":           func(in *usecase.SignupLegalEmployer) { in.RequiresActivityLicense = true },
		"missing attorney document": func(in *usecase.SignupLegalEmployer) { in.RepresentativePosition = enums.RepresentativeAttorney },
		"weak password":             func(in *usecase.SignupLegalEmployer) { in.Password = "123456789012" },
		"email":                     func(in *usecase.SignupLegalEmployer) { in.Email = "bad" },
		"website":                   func(in *usecase.SignupLegalEmployer) { in.Website = "ftp://example.com" },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			m := &legalEmployerMemory{}
			in := legalEmployerFixture()
			mutate(&in)
			if err := legalEmployerService(m).Signup(context.Background(), in); !errors.Is(err, domainerrors.ErrInvalidSignup) {
				t.Fatalf("invalid field returned %v", err)
			}
			if m.smsCount != 0 || m.putCount != 0 || m.legalPending != nil {
				t.Fatal("invalid information caused side effects")
			}
		})
	}
}

func TestLegalEmployerFailuresCleanNewFilesAndPreservePrevious(t *testing.T) {
	for _, failure := range []string{"media", "storage", "database", "sms"} {
		t.Run(failure, func(t *testing.T) {
			m := &legalEmployerMemory{}
			s := legalEmployerService(m)
			ctx := context.Background()
			if err := s.Signup(ctx, legalEmployerFixture()); err != nil {
				t.Fatal(err)
			}
			m.legalPending.Challenge.SentAt = time.Now().Add(-2 * time.Minute)
			oldKey, oldHash := m.legalPending.Registration.IdentityDocument.Key, m.legalPending.Challenge.CodeHash
			switch failure {
			case "media":
				m.mediaErr = domainerrors.ErrInvalidUpload
			case "storage":
				m.failPut = m.putCount + 2
			case "database":
				m.pendingErr = errors.New("database unavailable")
			case "sms":
				m.smsErr = errors.New("SMS unavailable")
			}
			if err := s.Signup(ctx, legalEmployerFixture()); err == nil {
				t.Fatal("failed replacement accepted")
			}
			if len(m.objects) != 3 || m.legalPending.Registration.IdentityDocument.Key != oldKey || m.legalPending.Challenge.CodeHash != oldHash {
				t.Fatal("failed replacement changed original pending submission or left new files")
			}
		})
	}
}

func TestLegalEmployerOTPLimitsAndResend(t *testing.T) {
	for _, scenario := range []string{"malformed", "expired code", "expired submission"} {
		t.Run(scenario, func(t *testing.T) {
			m := &legalEmployerMemory{}
			s := legalEmployerService(m)
			ctx := context.Background()
			in := legalEmployerFixture()
			if err := s.Signup(ctx, in); err != nil {
				t.Fatal(err)
			}
			m.legalPending.Challenge.SentAt = time.Now().Add(-2 * time.Minute)
			if err := s.RequestOTP(ctx, in.Phone); err != nil {
				t.Fatal(err)
			}
			if m.putCount != 3 || m.smsCount != 2 {
				t.Fatal("resend uploaded files or did not send code")
			}
			switch scenario {
			case "expired code":
				m.legalPending.Challenge.ExpiresAt = time.Now().Add(-time.Second)
			case "expired submission":
				m.legalPending.Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
			}
			for i := 0; i < 5; i++ {
				if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: "bad"}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
					t.Fatal(err)
				}
			}
			if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}); !errors.Is(err, domainerrors.ErrInvalidOTP) {
				t.Fatal("locked or expired code accepted")
			}
			if m.legal != nil {
				t.Fatal("account created without verification")
			}
		})
	}
}

func TestLegalEmployerLoginThrottling(t *testing.T) {
	m := &legalEmployerMemory{}
	s := legalEmployerService(m)
	ctx := context.Background()
	in := legalEmployerFixture()
	if err := s.Signup(ctx, in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.VerifyOTP(ctx, usecase.VerifySignupOTP{Phone: in.Phone, OTP: m.smsCode}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if _, err := s.Login(ctx, in.Phone, "WrongPassword123!"); !errors.Is(err, domainerrors.ErrInvalidCredentials) {
			t.Fatal(err)
		}
	}
	if _, err := s.Login(ctx, in.Phone, in.Password); !errors.Is(err, domainerrors.ErrLoginThrottled) {
		t.Fatal("login lockout bypassed")
	}
}
