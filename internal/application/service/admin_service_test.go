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
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/security"
)

type adminMemory struct {
	signupMemory
	admin                     *entities.Admin
	parent                    *entities.Parent
	filterStatus              string
	filterLimit, filterOffset int
}
type adminLoginMemory struct{ m *adminMemory }
type adminReviewMemory struct{ m *adminMemory }

func (m *adminMemory) Factory() ports.RepositoryFactory       { return m }
func (m *adminMemory) AdminRepository() ports.AdminRepository { return &adminLoginMemory{m} }
func (m *adminMemory) AdminReviewRepository() ports.AdminReviewRepository {
	return &adminReviewMemory{m}
}
func (m *adminMemory) WithTransaction(fn func(ports.RepositoryFactory) error) error {
	seeker, parent, attempt := cloneSeeker(m.seeker), cloneParent(m.parent), cloneAttempt(m.attempt)
	err := fn(m)
	if err == nil {
		err = m.commitErr
	}
	if err != nil {
		m.seeker, m.parent, m.attempt = seeker, parent, attempt
	}
	return err
}
func (*adminLoginMemory) LockUsername(context.Context, string) error { return nil }
func (r *adminLoginMemory) FindByUsername(_ context.Context, u string) (*entities.Admin, error) {
	if r.m.admin != nil && r.m.admin.Username == u {
		copy := *r.m.admin
		return &copy, nil
	}
	return nil, nil
}
func (r *adminLoginMemory) GetLoginAttempt(ctx context.Context, u string) (*entities.LoginAttempt, error) {
	return r.m.signupMemory.GetLoginAttempt(ctx, u)
}
func (r *adminLoginMemory) SaveLoginAttempt(ctx context.Context, a *entities.LoginAttempt) error {
	return r.m.signupMemory.SaveLoginAttempt(ctx, a)
}
func (r *adminLoginMemory) DeleteLoginAttempt(ctx context.Context, u string) error {
	return r.m.signupMemory.DeleteLoginAttempt(ctx, u)
}
func (r *adminReviewMemory) FindJobSeeker(_ context.Context, id uint64) (*entities.JobSeeker, error) {
	if r.m.seeker != nil && r.m.seeker.ID == id {
		return cloneSeeker(r.m.seeker), nil
	}
	return nil, nil
}
func (r *adminReviewMemory) FindParent(_ context.Context, id uint64) (*entities.Parent, error) {
	if r.m.parent != nil && r.m.parent.ID == id {
		return cloneParent(r.m.parent), nil
	}
	return nil, nil
}
func (r *adminReviewMemory) ReviewJobSeeker(_ context.Context, id uint64, review valueobjects.AdminReview) error {
	if r.m.seeker == nil || r.m.seeker.ID != id {
		return domainerrors.ErrReviewNotFound
	}
	r.m.seeker.AdminReview = review
	return nil
}
func (r *adminReviewMemory) ReviewParent(_ context.Context, id uint64, review valueobjects.AdminReview) error {
	if r.m.parent == nil || r.m.parent.ID != id {
		return domainerrors.ErrReviewNotFound
	}
	r.m.parent.AdminReview = review
	return nil
}
func (r *adminReviewMemory) ListJobSeekers(_ context.Context, status string, limit, offset int) ([]*entities.JobSeeker, int64, error) {
	r.m.filterStatus, r.m.filterLimit, r.m.filterOffset = status, limit, offset
	return []*entities.JobSeeker{cloneSeeker(r.m.seeker)}, 1, nil
}
func (r *adminReviewMemory) ListParents(_ context.Context, status string, limit, offset int) ([]*entities.Parent, int64, error) {
	r.m.filterStatus, r.m.filterLimit, r.m.filterOffset = status, limit, offset
	return []*entities.Parent{cloneParent(r.m.parent)}, 1, nil
}

type adminTokens struct{}

func (adminTokens) IssueAdminAccessToken(u string) (*valueobjects.AccessToken, error) {
	return &valueobjects.AccessToken{Value: "admin-token", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (adminTokens) VerifyAdminAccessToken(token string) (string, error) {
	if token != "admin-token" {
		return "", domainerrors.ErrUnauthenticated
	}
	return "admin", nil
}
func adminFixture(t *testing.T) (*adminMemory, *AdminService, *employerMemory) {
	t.Helper()
	m := &adminMemory{signupMemory: *verifiedMemory(t), admin: &entities.Admin{Username: "admin", PasswordHash: "hash:admin-password"}, parent: &entities.Parent{ID: 7, FirstName: "Parent", Phone: "09129876543", AdminReview: valueobjects.AdminReview{AdminApprovalStatus: enums.AdminReviewPending}, IdentityDocument: &valueobjects.StoredObject{Key: "parents/identity.png"}, ConsentVideo: &valueobjects.StoredObject{Key: "parents/video.mp4"}}}
	id := m.parent.ID
	m.seeker.ParentID = &id
	storage := &employerMemory{objects: map[string]valueobjects.StoredObject{}}
	storage.objects[m.parent.IdentityDocument.Key] = *m.parent.IdentityDocument
	storage.objects[m.parent.ConsentVideo.Key] = *m.parent.ConsentVideo
	storage.objects[m.seeker.VerificationVideo.Key] = *m.seeker.VerificationVideo
	return m, NewAdminService(m, testPasswords{}, adminTokens{}, storage), storage
}
func TestAdminLoginAndThrottle(t *testing.T) {
	m, s, _ := adminFixture(t)
	ctx := context.Background()
	login, err := s.Login(ctx, " admin ", "admin-password")
	if err != nil || login.Username != "admin" || login.AccessToken.Value != "admin-token" {
		t.Fatal("admin login failed", err)
	}
	if _, err := s.Login(ctx, "missing", "admin-password"); !errors.Is(err, domainerrors.ErrInvalidAdminCredentials) {
		t.Fatal("unknown admin accepted")
	}
	for i := 0; i < maxLoginFailures; i++ {
		if _, err := s.Login(ctx, "admin", "wrong-password"); !errors.Is(err, domainerrors.ErrInvalidAdminCredentials) {
			t.Fatal(err)
		}
	}
	if _, err := s.Login(ctx, "admin", "admin-password"); !errors.Is(err, domainerrors.ErrLoginThrottled) {
		t.Fatal("admin login not throttled", err)
	}
	m.attempt.WindowStartedAt = time.Now().Add(-16 * time.Minute)
	if _, err := s.Login(ctx, "admin", "admin-password"); err != nil || m.attempt != nil {
		t.Fatal("successful login did not clear throttle", err)
	}
	if _, err := s.Authenticate(ctx, "user-token"); !errors.Is(err, domainerrors.ErrUnauthenticated) {
		t.Fatal("user token authorized admin")
	}
	if u, err := s.Authenticate(ctx, "admin-token"); err != nil || u != "admin" {
		t.Fatal(err)
	}
	m.admin = nil
	if _, err := s.Authenticate(ctx, "admin-token"); !errors.Is(err, domainerrors.ErrUnauthenticated) {
		t.Fatal("deleted admin token accepted")
	}
}

func TestAdminLoginAcceptsProvisionedShortPassword(t *testing.T) {
	m, s, _ := adminFixture(t)
	hasher, err := security.NewBcryptPasswordHasher()
	if err != nil {
		t.Fatal(err)
	}
	m.admin.PasswordHash, err = hasher.Hash("admin")
	if err != nil {
		t.Fatal(err)
	}
	s.passwords = hasher
	if _, err := s.Login(context.Background(), "admin", "admin"); err != nil {
		t.Fatalf("matching SQL-provisioned password rejected: %v", err)
	}
	if _, err := s.Login(context.Background(), "admin", "wrong"); !errors.Is(err, domainerrors.ErrInvalidAdminCredentials) {
		t.Fatal("wrong short password accepted", err)
	}
	if m.attempt == nil || m.attempt.Failures != 1 {
		t.Fatal("wrong short passwords must count toward login throttling")
	}
	for _, password := range []string{"", strings.Repeat("x", 73)} {
		if _, err := s.Login(context.Background(), "admin", password); !errors.Is(err, domainerrors.ErrInvalidAdminCredentials) {
			t.Fatal("invalid password length accepted", err)
		}
	}
}
func TestAdminReviewsAreIndependentAndMessagesOptional(t *testing.T) {
	m, s, _ := adminFixture(t)
	ctx := context.Background()
	result, err := s.ReviewJobSeeker(ctx, 1, usecase.AdminReviewDecision{Approved: true, Reviewer: "admin"})
	if err != nil || result.AdminApprovalStatus != enums.AdminReviewApproved || result.AdminReviewedAt == nil || result.AdminReviewedBy != "admin" {
		t.Fatal("job seeker not approved", err)
	}
	if m.parent.AdminApprovalStatus != enums.AdminReviewPending {
		t.Fatal("child approval changed parent review")
	}
	result, err = s.ReviewJobSeeker(ctx, 1, usecase.AdminReviewDecision{Approved: false, Message: "Please submit a clearer video", Reviewer: "admin"})
	if err != nil || result.AdminApprovalStatus != enums.AdminReviewRejected || result.AdminReviewMessage != "Please submit a clearer video" {
		t.Fatal("rejection message lost", err)
	}
	parent, err := s.ReviewParent(ctx, 7, usecase.AdminReviewDecision{Approved: false, Reviewer: "admin"})
	if err != nil || parent.AdminReviewMessage != "Parent was not approved" {
		t.Fatal("default rejection message missing", err)
	}
	if m.seeker.AdminApprovalStatus != enums.AdminReviewRejected {
		t.Fatal("parent review changed child status")
	}
	parent, err = s.ReviewParent(ctx, 7, usecase.AdminReviewDecision{Approved: true, Reviewer: "admin"})
	if err != nil || parent.AdminApprovalStatus != enums.AdminReviewApproved || parent.AdminReviewMessage != "" {
		t.Fatal("approval did not clear old rejection", err)
	}
}
func TestAdminCannotApproveIncompleteSignup(t *testing.T) {
	for _, scenario := range []string{"minor missing parent", "missing account", "unknown admin"} {
		t.Run(scenario, func(t *testing.T) {
			m, s, _ := adminFixture(t)
			id := uint64(1)
			switch scenario {
			case "minor missing parent":
				m.seeker.ParentID = nil
				m.seeker.BirthDate = valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 16, Month: 1, Day: 1}
			case "missing account":
				id = 999
			case "unknown admin":
				m.admin = nil
			}
			if _, err := s.ReviewJobSeeker(context.Background(), id, usecase.AdminReviewDecision{Approved: true, Reviewer: "admin"}); err == nil {
				t.Fatal("invalid approval accepted")
			}
			if m.seeker.AdminApprovalStatus == enums.AdminReviewApproved {
				t.Fatal("failed approval changed status")
			}
		})
	}
}
func TestAdminReviewRollbackAndAdultSkip(t *testing.T) {
	m, s, _ := adminFixture(t)
	ctx := context.Background()
	m.commitErr = errors.New("commit failed")
	if _, err := s.ReviewJobSeeker(ctx, 1, usecase.AdminReviewDecision{Approved: true, Reviewer: "admin"}); err == nil || m.seeker.AdminApprovalStatus == enums.AdminReviewApproved {
		t.Fatal("failed commit approved seeker")
	}
	if _, err := s.ReviewParent(ctx, 7, usecase.AdminReviewDecision{Approved: true, Reviewer: "admin"}); err == nil || m.parent.AdminApprovalStatus != enums.AdminReviewPending {
		t.Fatal("failed commit approved parent")
	}
	m.commitErr = nil
	m.seeker.ParentID = nil
	m.seeker.BirthDate = valueobjects.JalaliDate{Year: time.Now().Year() - 621 - 20, Month: 1, Day: 1}
	if _, err := s.ReviewJobSeeker(ctx, 1, usecase.AdminReviewDecision{Approved: true, Reviewer: "admin"}); err != nil {
		t.Fatal("adult without a parent could not be approved", err)
	}
	if _, err := s.ReviewParent(ctx, 7, usecase.AdminReviewDecision{Approved: false, Reviewer: "admin", Message: strings.Repeat("x", 1001)}); !errors.Is(err, domainerrors.ErrInvalidReview) {
		t.Fatal("oversized review accepted")
	}
}
func TestAdminListFiltersAndPrivateDocuments(t *testing.T) {
	m, s, _ := adminFixture(t)
	ctx := context.Background()
	if _, _, err := s.JobSeekers(ctx, usecase.AdminListFilter{}); err != nil || m.filterStatus != "pending" || m.filterLimit != 20 {
		t.Fatal("incorrect review defaults", err)
	}
	if _, _, err := s.Parents(ctx, usecase.AdminListFilter{}); err != nil || m.filterStatus != "pending" {
		t.Fatal("incorrect parent review default", err)
	}
	if _, _, err := s.JobSeekers(ctx, usecase.AdminListFilter{Status: "accepted"}); !errors.Is(err, domainerrors.ErrInvalidReview) {
		t.Fatal("duplicate job seeker approval alias accepted", err)
	}
	if _, _, err := s.Parents(ctx, usecase.AdminListFilter{Status: "accepted"}); !errors.Is(err, domainerrors.ErrInvalidReview) {
		t.Fatal("duplicate parent approval alias accepted", err)
	}
	if _, _, err := s.JobSeekers(ctx, usecase.AdminListFilter{Status: "approved"}); err != nil || m.filterStatus != "approved" {
		t.Fatal("job seeker accepted filter failed", err)
	}
	if _, _, err := s.Parents(ctx, usecase.AdminListFilter{Status: "approved"}); err != nil || m.filterStatus != "approved" {
		t.Fatal("parent approved filter failed", err)
	}
	if _, _, err := s.JobSeekers(ctx, usecase.AdminListFilter{Status: "unapproved"}); !errors.Is(err, domainerrors.ErrInvalidReview) {
		t.Fatal("removed job seeker filter accepted", err)
	}
	if _, _, err := s.Parents(ctx, usecase.AdminListFilter{Status: "unapproved"}); !errors.Is(err, domainerrors.ErrInvalidReview) {
		t.Fatal("removed parent filter accepted", err)
	}
	if _, _, err := s.Parents(ctx, usecase.AdminListFilter{Status: "rejected", Limit: 10, Offset: 5}); err != nil || m.filterStatus != "rejected" || m.filterOffset != 5 {
		t.Fatal("filter lost", err)
	}
	for _, f := range []usecase.AdminListFilter{{Status: "bad"}, {Limit: 101}, {Offset: -1}} {
		if _, _, err := s.JobSeekers(ctx, f); !errors.Is(err, domainerrors.ErrInvalidReview) {
			t.Fatal("invalid filter accepted")
		}
	}
	if _, err := s.JobSeekerVideoURL(ctx, 999); !errors.Is(err, domainerrors.ErrReviewNotFound) {
		t.Fatal("missing seeker received URL")
	}
	if _, err := s.ParentDocumentURL(ctx, 7, "unknown"); !errors.Is(err, domainerrors.ErrInvalidReview) {
		t.Fatal("unknown document accepted")
	}
	for _, kind := range []string{"identity", "consent-video"} {
		if _, err := s.ParentDocumentURL(ctx, 7, kind); err != nil {
			t.Fatal(err)
		}
	}
}
