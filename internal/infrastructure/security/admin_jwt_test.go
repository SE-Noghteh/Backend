package security

import (
	"strings"
	"testing"
	"time"
	"zedlancer/backend/internal/domain/enums"
)

func TestAdminJWTIsolatedFromUserRoles(t *testing.T) {
	m, err := NewJWTManager(strings.Repeat("s", 32), "test", "test", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	token, err := m.IssueAdminAccessToken("review-admin")
	if err != nil {
		t.Fatal(err)
	}
	username, err := m.VerifyAdminAccessToken(token.Value)
	if err != nil || username != "review-admin" {
		t.Fatal("admin identity lost", err)
	}
	for _, role := range []enums.AccountRole{enums.RoleJobSeeker, enums.RoleIndividualEmployer, enums.RoleLegalEmployer} {
		user, err := m.IssueAccessToken(1, role)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := m.VerifyAdminAccessToken(user.Value); err == nil {
			t.Fatal("user token accepted as admin", role)
		}
		if _, err := m.VerifyAccessToken(token.Value, role); err == nil {
			t.Fatal("admin token accepted as user", role)
		}
	}
	other, _ := NewJWTManager(strings.Repeat("x", 32), "test", "test", time.Hour)
	if _, err := other.VerifyAdminAccessToken(token.Value); err == nil {
		t.Fatal("wrong signature accepted")
	}
	if _, err := m.VerifyAdminAccessToken("bad-token"); err == nil {
		t.Fatal("invalid token accepted")
	}
	for _, username := range []string{"", " admin", strings.Repeat("a", 101)} {
		if _, err := m.IssueAdminAccessToken(username); err == nil {
			t.Fatal("invalid admin identity accepted")
		}
	}
}
