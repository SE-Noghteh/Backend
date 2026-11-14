package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"strings"
	"time"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
)

func (m *JWTManager) IssueAdminAccessToken(username string) (*valueobjects.AccessToken, error) {
	if username == "" || len(username) > 100 || strings.TrimSpace(username) != username {
		return nil, fmt.Errorf("invalid admin identity")
	}
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(m.ttl)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	claims := accessClaims{Role: string(enums.RoleAdmin), RegisteredClaims: jwt.RegisteredClaims{Issuer: m.issuer, Audience: jwt.ClaimStrings{m.audience}, Subject: username, IssuedAt: jwt.NewNumericDate(now), NotBefore: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(expires), ID: hex.EncodeToString(nonce)}}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return nil, err
	}
	return &valueobjects.AccessToken{Value: value, ExpiresAt: expires}, nil
}
func (m *JWTManager) VerifyAdminAccessToken(value string) (string, error) {
	if len(value) > 4096 {
		return "", domainerrors.ErrUnauthenticated
	}
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(value, claims, func(_ *jwt.Token) (any, error) { return m.secret, nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithIssuer(m.issuer), jwt.WithAudience(m.audience), jwt.WithIssuedAt())
	if err != nil || !token.Valid || claims.Role != string(enums.RoleAdmin) || claims.IssuedAt == nil || claims.ID == "" || claims.Subject == "" || len(claims.Subject) > 100 || strings.TrimSpace(claims.Subject) != claims.Subject {
		return "", domainerrors.ErrUnauthenticated
	}
	return claims.Subject, nil
}
