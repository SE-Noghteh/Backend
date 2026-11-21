package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/domain/valueobjects"
)

type parentMemory struct {
	signupMemory
	parents     map[string]*entities.Parent
	invitations map[string]*entities.ParentInvitation
	pending     map[string]*entities.PendingParentSignup
}
type memoryParentRepository struct{ m *parentMemory }

func (m *parentMemory) Factory() ports.RepositoryFactory         { return m }
func (m *parentMemory) ParentRepository() ports.ParentRepository { return &memoryParentRepository{m} }
func cloneParent(p *entities.Parent) *entities.Parent {
	if p == nil {
		return nil
	}
	out := *p
	return &out
}
func cloneParentPending(p *entities.PendingParentSignup) *entities.PendingParentSignup {
	if p == nil {
		return nil
	}
	out := *p
	out.Registration = cloneParent(p.Registration)
	out.Challenge = cloneOTP(p.Challenge)
	return &out
}
func (m *parentMemory) WithTransaction(fn func(ports.RepositoryFactory) error) error {
	seeker := cloneSeeker(m.seeker)
	parents := map[string]*entities.Parent{}
	for k, v := range m.parents {
		parents[k] = cloneParent(v)
	}
	invitations := map[string]*entities.ParentInvitation{}
	for k, v := range m.invitations {
		copy := *v
		invitations[k] = &copy
	}
	pending := map[string]*entities.PendingParentSignup{}
	for k, v := range m.pending {
		pending[k] = cloneParentPending(v)
	}
	err := fn(m)
	if err == nil {
		err = m.commitErr
	}
	if err != nil {
		m.seeker, m.parents, m.invitations, m.pending = seeker, parents, invitations, pending
	}
	return err
}
func (*memoryParentRepository) LockPhone(context.Context, string) error { return nil }
func (r *memoryParentRepository) FindByPhone(_ context.Context, phone string) (*entities.Parent, error) {
	return cloneParent(r.m.parents[phone]), nil
}
func (r *memoryParentRepository) Create(_ context.Context, p *entities.Parent) error {
	if r.m.parents[p.Phone] != nil {
		return domainerrors.ErrSignupConflict
	}
	p.ID = uint64(len(r.m.parents) + 1)
	r.m.parents[p.Phone] = cloneParent(p)
	return nil
}
func (r *memoryParentRepository) SaveInvitation(ctx context.Context, i *entities.ParentInvitation) error {
	_ = r.DeleteInvitation(ctx, i.JobSeekerID)
	copy := *i
	r.m.invitations[i.CodeHash] = &copy
	return nil
}
func (r *memoryParentRepository) FindInvitation(_ context.Context, hash string) (*entities.ParentInvitation, error) {
	i := r.m.invitations[hash]
	if i == nil {
		return nil, nil
	}
	copy := *i
	return &copy, nil
}
func (r *memoryParentRepository) DeleteInvitation(_ context.Context, id uint64) error {
	for k, v := range r.m.invitations {
		if v.JobSeekerID == id {
			delete(r.m.invitations, k)
		}
	}
	return nil
}
func (r *memoryParentRepository) GetPending(_ context.Context, phone string) (*entities.PendingParentSignup, error) {
	return cloneParentPending(r.m.pending[phone]), nil
}
func (r *memoryParentRepository) SavePending(_ context.Context, p *entities.PendingParentSignup) error {
	if r.m.saveErr != nil {
		return r.m.saveErr
	}
	r.m.pending[p.Registration.Phone] = cloneParentPending(p)
	return nil
}
func (r *memoryParentRepository) DeletePending(_ context.Context, phone string) error {
	delete(r.m.pending, phone)
	return nil
}
func (r *memoryParentRepository) SetParentConnection(_ context.Context, id uint64, parentID *uint64, relationship string) error {
	if r.m.seeker == nil || r.m.seeker.ID != id {
		return domainerrors.ErrUnauthenticated
	}
	r.m.seeker.ParentID, r.m.seeker.ParentRelationship = parentID, relationship
	return nil
}
func parentFixture(t *testing.T) (*parentMemory, *ParentService, *employerMemory) {
	t.Helper()
	m := &parentMemory{signupMemory: *verifiedMemory(t), parents: map[string]*entities.Parent{}, invitations: map[string]*entities.ParentInvitation{}, pending: map[string]*entities.PendingParentSignup{}}
	m.seeker.BirthDate = valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 16, Month: 1, Day: 1}
	m.smsDestination = "989129876543"
	storage := &employerMemory{}
	s := NewParentService(m, &m.signupMemory, testPasswords{}, storage, storage)
	return m, s, storage
}
func parentInvitation(t *testing.T, s *ParentService) string {
	t.Helper()
	r, err := s.InviteParent(context.Background(), 1, usecase.JobSeekerParentInput{FirstName: " Parent ", LastName: " Family ", Phone: "+989129876543"})
	if err != nil {
		t.Fatal(err)
	}
	if r.SignupStatus != valueobjects.JobSeekerPending || r.ParentID != nil || len(r.InvitationCode) != 8 || r.ExpiresAt == nil || time.Until(*r.ExpiresAt) < 7*24*time.Hour-time.Minute {
		t.Fatalf("invalid invitation response: %+v", r)
	}
	return r.InvitationCode
}
func parentSignup(code string) usecase.SignupParent {
	return usecase.SignupParent{Code: code, Relationship: "father", Password: "parent-password", ConsentAccepted: true, IdentityDocument: ports.UploadFile{Name: "identity.png", Body: strings.NewReader("image")}, ConsentVideo: ports.UploadFile{Name: "consent.mp4", Body: strings.NewReader("video")}}
}

func TestParentInvitationSignupAndSecondChild(t *testing.T) {
	m, s, storage := parentFixture(t)
	ctx := context.Background()
	code := parentInvitation(t, s)
	if m.invitations[code] != nil || len(m.parents) != 0 {
		t.Fatal("raw code stored or parent created early")
	}
	info, err := s.ValidateInvitation(ctx, code)
	if err != nil || info.FirstName != "Parent" || info.LastName != "Family" || info.Phone != changedPhone {
		t.Fatalf("saved identity missing: %+v %v", info, err)
	}
	if err := s.Signup(ctx, parentSignup(code)); err != nil {
		t.Fatal(err)
	}
	if len(m.parents) != 0 || m.seeker.ParentID != nil || len(storage.objects) != 2 {
		t.Fatal("signup created parent or failed to save both uploads")
	}
	if m.pending[changedPhone].Registration.PasswordHash != "hash:parent-password" {
		t.Fatal("password was not hashed")
	}
	parent, err := s.VerifyOTP(ctx, changedPhone, m.smsCode)
	if err != nil {
		t.Fatal(err)
	}
	if parent.FirstName != "Parent" || parent.Phone != changedPhone || parent.PhoneVerifiedAt.IsZero() || m.seeker.ParentID == nil || *m.seeker.ParentID != parent.ID || m.seeker.ParentRelationship != "father" {
		t.Fatal("parent was not created and linked")
	}
	if len(m.pending) != 0 || len(m.invitations) != 0 {
		t.Fatal("successful verification did not consume challenge/code")
	}
	if _, err := s.VerifyOTP(ctx, changedPhone, m.smsCode); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("invitation replay accepted")
	}
	firstChild := cloneSeeker(m.seeker)
	m.seeker = cloneSeeker(firstChild)
	m.seeker.ID = 2
	m.seeker.ParentID = nil
	r, err := s.LinkRegisteredParent(ctx, 2, changedPhone)
	if err != nil || r.InvitationCode != "" || r.ParentID == nil || *r.ParentID != parent.ID || r.SignupStatus != valueobjects.JobSeekerPending || len(m.parents) != 1 || *firstChild.ParentID != *m.seeker.ParentID {
		t.Fatalf("existing parent was not reused: %+v %v", r, err)
	}
	if m.parents[changedPhone].FirstName != "Parent" {
		t.Fatal("second child overwrote parent identity")
	}
}

func TestParentInvitationRequiresDetailsAtAnyAge(t *testing.T) {
	for _, age := range []int{16, 20} {
		t.Run(fmt.Sprint(age), func(t *testing.T) {
			m, s, _ := parentFixture(t)
			m.seeker.BirthDate = valueobjects.JalaliDate{Year: time.Now().Year() - 621 - age, Month: 1, Day: 1}
			code := parentInvitation(t, s)
			if _, err := s.InviteParent(context.Background(), 1, usecase.JobSeekerParentInput{}); !errors.Is(err, domainerrors.ErrInvalidParent) {
				t.Fatal("invitation without parent details accepted", err)
			}
			if _, err := s.ValidateInvitation(context.Background(), code); err != nil {
				t.Fatal("invalid request removed existing invitation", err)
			}
		})
	}
}

func TestConnectedParentBlocksBothParentApprovalAPIs(t *testing.T) {
	for _, flow := range []string{"registered", "invite"} {
		t.Run(flow, func(t *testing.T) {
			m, s, _ := parentFixture(t)
			id := uint64(9)
			m.seeker.ParentID = &id
			m.seeker.ParentRelationship = "mother"
			m.parents[changedPhone] = &entities.Parent{ID: 10, Phone: changedPhone, PhoneVerifiedAt: time.Now().UTC()}
			var err error
			switch flow {
			case "registered":
				_, err = s.LinkRegisteredParent(context.Background(), 1, changedPhone)
			case "invite":
				_, err = s.InviteParent(context.Background(), 1, usecase.JobSeekerParentInput{FirstName: "New", LastName: "Parent", Phone: "09121111111"})
			}
			if !errors.Is(err, domainerrors.ErrParentAlreadyConnected) {
				t.Fatalf("expected already connected, got %v", err)
			}
			if len(m.invitations) != 0 || m.seeker.ParentID == nil || *m.seeker.ParentID != 9 || m.seeker.ParentRelationship != "mother" {
				t.Fatal("existing connection was changed or invitation generated")
			}
		})
	}
}

func TestParentInvitationExpiryReplacementAndDeletedSeeker(t *testing.T) {
	for _, scenario := range []string{"expired", "replaced", "deleted", "missing", "pending expired"} {
		t.Run(scenario, func(t *testing.T) {
			m, s, storage := parentFixture(t)
			ctx := context.Background()
			code := parentInvitation(t, s)
			if scenario == "pending expired" {
				if err := s.Signup(ctx, parentSignup(code)); err != nil {
					t.Fatal(err)
				}
				m.pending[changedPhone].Registration.CreatedAt = time.Now().Add(-31 * time.Minute)
			}
			switch scenario {
			case "expired":
				m.invitations[invitationHash(code)].ExpiresAt = time.Now().Add(-time.Second)
			case "replaced":
				_ = parentInvitation(t, s)
			case "deleted":
				m.seeker = nil
			case "missing":
				code = strings.Repeat("x", 8)
			}
			if scenario != "pending expired" {
				if _, err := s.ValidateInvitation(ctx, code); !errors.Is(err, domainerrors.ErrInvalidInvitation) {
					t.Fatal("invalid code validated", err)
				}
				if err := s.Signup(ctx, parentSignup(code)); !errors.Is(err, domainerrors.ErrInvalidInvitation) {
					t.Fatal("invalid code signed up", err)
				}
				if storage.putCount != 0 {
					t.Fatal("invalid code uploaded files")
				}
			}
			if _, err := s.VerifyOTP(ctx, changedPhone, m.smsCode); err == nil {
				t.Fatal("invalid invitation/pending signup accepted")
			}
			if len(m.parents) != 0 {
				t.Fatal("invalid signup created parent")
			}
		})
	}
}

func TestParentSignupValidation(t *testing.T) {
	for _, scenario := range []string{"no code", "guardian", "password", "consent", "image missing", "video missing", "pdf"} {
		t.Run(scenario, func(t *testing.T) {
			m, s, storage := parentFixture(t)
			code := parentInvitation(t, s)
			in := parentSignup(code)
			switch scenario {
			case "no code":
				in.Code = ""
			case "guardian":
				in.Relationship = "guardian"
			case "password":
				in.Password = "short"
			case "consent":
				in.ConsentAccepted = false
			case "image missing":
				in.IdentityDocument.Body = nil
			case "video missing":
				in.ConsentVideo.Body = nil
			case "pdf":
				in.IdentityDocument.Name = "identity.pdf"
			}
			if err := s.Signup(context.Background(), in); err == nil {
				t.Fatal("invalid submission accepted")
			}
			if storage.putCount != 0 || len(m.parents) != 0 || len(m.pending) != 0 {
				t.Fatal("invalid submission had side effects")
			}
		})
	}
}

func TestParentOTPResendAttemptsAndExpiry(t *testing.T) {
	m, s, storage := parentFixture(t)
	ctx := context.Background()
	code := parentInvitation(t, s)
	if err := s.ResendOTP(ctx, code); err == nil {
		t.Fatal("resend without signup accepted")
	}
	if err := s.Signup(ctx, parentSignup(code)); err != nil {
		t.Fatal(err)
	}
	if err := s.ResendOTP(ctx, code); !errors.Is(err, domainerrors.ErrOTPTooSoon) {
		t.Fatal("resend cooldown bypassed")
	}
	for i := 0; i < maxOTPAttempts; i++ {
		if _, err := s.VerifyOTP(ctx, changedPhone, "bad"); !errors.Is(err, domainerrors.ErrInvalidOTP) {
			t.Fatal(err)
		}
	}
	if m.pending[changedPhone].Challenge.Attempts != maxOTPAttempts {
		t.Fatal("failed attempts were rolled back")
	}
	if _, err := s.VerifyOTP(ctx, changedPhone, m.smsCode); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("locked code accepted")
	}
	m.pending[changedPhone].Challenge.SentAt = time.Now().Add(-2 * time.Minute)
	if err := s.ResendOTP(ctx, code); err != nil {
		t.Fatal(err)
	}
	if storage.putCount != 2 {
		t.Fatal("resend reuploaded documents")
	}
	m.pending[changedPhone].Challenge.ExpiresAt = time.Now().Add(-time.Second)
	if _, err := s.VerifyOTP(ctx, changedPhone, m.smsCode); !errors.Is(err, domainerrors.ErrInvalidOTP) {
		t.Fatal("expired OTP accepted")
	}
}

func TestParentSignupFailuresAndVerificationRollback(t *testing.T) {
	for _, scenario := range []string{"sms", "save", "storage", "media", "commit", "verify commit"} {
		t.Run(scenario, func(t *testing.T) {
			m, s, storage := parentFixture(t)
			ctx := context.Background()
			code := parentInvitation(t, s)
			switch scenario {
			case "sms":
				m.smsErr = errors.New("SMS failed")
			case "save":
				m.saveErr = errors.New("save failed")
			case "storage":
				storage.failPut = 2
			case "media":
				storage.mediaErr = domainerrors.ErrInvalidUpload
			case "commit":
				m.commitErr = errors.New("commit failed")
			}
			if scenario == "verify commit" {
				if err := s.Signup(ctx, parentSignup(code)); err != nil {
					t.Fatal(err)
				}
				m.commitErr = errors.New("verify failed")
				if _, err := s.VerifyOTP(ctx, changedPhone, m.smsCode); err == nil {
					t.Fatal("failed verification accepted")
				}
				if len(m.parents) != 0 || m.seeker.ParentID != nil || len(m.pending) != 1 || len(m.invitations) != 1 || len(storage.objects) != 2 {
					t.Fatal("verification did not roll back")
				}
				return
			}
			if err := s.Signup(ctx, parentSignup(code)); err == nil {
				t.Fatal("failed signup accepted")
			}
			if len(m.parents) != 0 || len(m.pending) != 0 || len(storage.objects) != 0 || m.seeker.ParentID != nil {
				t.Fatal("failed signup left data or uploads")
			}
		})
	}
}

func TestParentApprovalEndpointsDoNotSwitchFlows(t *testing.T) {
	m, s, _ := parentFixture(t)
	ctx := context.Background()
	if _, err := s.LinkRegisteredParent(ctx, 1, changedPhone); !errors.Is(err, domainerrors.ErrParentNotRegistered) {
		t.Fatalf("unregistered phone accepted: %v", err)
	}
	if len(m.invitations) != 0 || m.seeker.ParentID != nil {
		t.Fatal("registered endpoint created invitation or link for a new parent")
	}
	m.parents[changedPhone] = &entities.Parent{ID: 9, Phone: changedPhone, FirstName: "Saved", LastName: "Parent", Relationship: "mother", PhoneVerifiedAt: time.Now().UTC()}
	if _, err := s.InviteParent(ctx, 1, usecase.JobSeekerParentInput{FirstName: "New", LastName: "Name", Phone: changedPhone}); !errors.Is(err, domainerrors.ErrParentAlreadyRegistered) {
		t.Fatalf("invitation silently linked existing parent: %v", err)
	}
	if len(m.invitations) != 0 || m.seeker.ParentID != nil {
		t.Fatal("wrong endpoint changed state")
	}
	r, err := s.LinkRegisteredParent(ctx, 1, "+989129876543")
	if err != nil || r.ParentID == nil || *r.ParentID != 9 || r.InvitationCode != "" || r.ExpiresAt != nil || len(m.invitations) != 0 {
		t.Fatalf("registered link created a code or required names: %+v %v", r, err)
	}
}

func TestShortParentInvitationAndLegacyCode(t *testing.T) {
	m, s, _ := parentFixture(t)
	code := parentInvitation(t, s)
	if strings.Trim(code, "ABCDEFGHJKLMNPQRSTUVWXYZ23456789") != "" {
		t.Fatalf("unexpected characters: %s", code)
	}
	if _, err := s.ValidateInvitation(context.Background(), " "+strings.ToLower(code)+" "); err != nil {
		t.Fatal("lowercase short code rejected", err)
	}
	legacy := strings.Repeat("aB3_", 10) + "xYz"
	i := m.invitations[invitationHash(code)]
	delete(m.invitations, invitationHash(code))
	i.CodeHash = invitationHash(legacy)
	m.invitations[i.CodeHash] = i
	if _, err := s.ValidateInvitation(context.Background(), legacy); err != nil {
		t.Fatal("previous long code invalidated", err)
	}
}

func TestParentPhoneVerificationChecksSavedInvitation(t *testing.T) {
	for _, scenario := range []string{"expired", "replaced", "deleted"} {
		t.Run(scenario, func(t *testing.T) {
			m, s, _ := parentFixture(t)
			ctx := context.Background()
			code := parentInvitation(t, s)
			if err := s.Signup(ctx, parentSignup(code)); err != nil {
				t.Fatal(err)
			}
			switch scenario {
			case "expired":
				m.invitations[invitationHash(code)].ExpiresAt = time.Now().Add(-time.Second)
			case "replaced":
				_ = parentInvitation(t, s)
			case "deleted":
				m.seeker = nil
			}
			if _, err := s.VerifyOTP(ctx, "+989129876543", m.smsCode); !errors.Is(err, domainerrors.ErrInvalidInvitation) {
				t.Fatalf("invalid saved invitation accepted: %v", err)
			}
			if len(m.parents) != 0 {
				t.Fatal("phone verification bypassed invitation")
			}
		})
	}
}
