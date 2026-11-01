package security

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"zedlancer/backend/internal/domain/ports"
)

func testJWTManager(t *testing.T) *JWTManager {
	t.Helper()
	m, err := NewJWTManager(strings.Repeat("a", 32), "issuer", "audience", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestJWTIssueAndVerify(t *testing.T) {
	m := testJWTManager(t)
	token, err := m.IssueAccessToken(42)
	if err != nil {
		t.Fatal(err)
	}
	principal, err := m.VerifyAccessToken(token.Value)
	if err != nil || principal.ID != 42 || len(principal.TokenID) != 32 || !principal.ExpiresAt.Equal(token.ExpiresAt) {
		t.Fatalf("token failed validation: %v", err)
	}
	if token.ExpiresAt.Before(time.Now().Add(59 * time.Minute)) {
		t.Fatal("unexpected expiry")
	}
	if _, err := m.IssueAccessToken(0); err == nil {
		t.Fatal("zero identity accepted")
	}
}

func TestJWTRejectsInvalidClaimsAndAlgorithms(t *testing.T) {
	m := testJWTManager(t)
	for _, scenario := range []string{"expired", "missing expiration", "issuer", "audience", "missing issued at", "future issued at", "future not before", "role", "subject", "missing token id", "wrong key", "HS384", "none"} {
		t.Run(scenario, func(t *testing.T) {
			now := time.Now().UTC()
			claims := accessClaims{Role: "job_seeker", RegisteredClaims: jwt.RegisteredClaims{Issuer: "issuer", Audience: jwt.ClaimStrings{"audience"}, Subject: "42", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)), ID: "token-id"}}
			var method jwt.SigningMethod = jwt.SigningMethodHS256
			var key any = m.secret
			switch scenario {
			case "expired":
				claims.ExpiresAt = jwt.NewNumericDate(now.Add(-time.Minute))
			case "missing expiration":
				claims.ExpiresAt = nil
			case "issuer":
				claims.Issuer = "other"
			case "audience":
				claims.Audience = jwt.ClaimStrings{"other"}
			case "missing issued at":
				claims.IssuedAt = nil
			case "future issued at":
				claims.IssuedAt = jwt.NewNumericDate(now.Add(time.Hour))
			case "future not before":
				claims.NotBefore = jwt.NewNumericDate(now.Add(time.Hour))
			case "role":
				claims.Role = "employer"
			case "subject":
				claims.Subject = "0"
			case "missing token id":
				claims.ID = ""
			case "wrong key":
				key = []byte(strings.Repeat("b", 32))
			case "HS384":
				method = jwt.SigningMethodHS384
			case "none":
				method = jwt.SigningMethodNone
				key = jwt.UnsafeAllowNoneSignatureType
			}
			token, err := jwt.NewWithClaims(method, claims).SignedString(key)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := m.VerifyAccessToken(token); !errors.Is(err, ports.ErrUnauthenticated) {
				t.Fatalf("accepted invalid token: %v", err)
			}
		})
	}
	for _, value := range []string{"", "garbage", strings.Repeat("a", 4097)} {
		if _, err := m.VerifyAccessToken(value); !errors.Is(err, ports.ErrUnauthenticated) {
			t.Fatal("malformed token accepted")
		}
	}
}

func TestJWTRejectsInvalidConfiguration(t *testing.T) {
	for _, config := range []struct {
		secret, issuer, audience string
		ttl                      time.Duration
	}{
		{"short", "issuer", "audience", time.Hour},
		{strings.Repeat("a", 32), "", "audience", time.Hour},
		{strings.Repeat("a", 32), "issuer", "", time.Hour},
		{strings.Repeat("a", 32), "issuer", "audience", 0},
		{strings.Repeat("a", 32), "issuer", "audience", 25 * time.Hour},
	} {
		if _, err := NewJWTManager(config.secret, config.issuer, config.audience, config.ttl); err == nil {
			t.Fatal("invalid JWT configuration accepted")
		}
	}
}
