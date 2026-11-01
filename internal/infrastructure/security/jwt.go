package security

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"zedlancer/backend/bootstrap"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

type JWTManager struct {
	secret           []byte
	issuer, audience string
	ttl              time.Duration
}
type accessClaims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func ProvideJWTManager() (*JWTManager, error) {
	config := bootstrap.Run().Env.JWT
	ttl, err := time.ParseDuration(config.AccessTokenTTL)
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TOKEN_TTL must be a duration such as 1h")
	}
	return NewJWTManager(config.Secret, config.Issuer, config.Audience, ttl)
}

func NewJWTManager(secret, issuer, audience string, ttl time.Duration) (*JWTManager, error) {
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must contain at least 32 bytes")
	}
	if strings.TrimSpace(issuer) == "" || strings.TrimSpace(audience) == "" || ttl < time.Minute || ttl > 24*time.Hour {
		return nil, fmt.Errorf("JWT issuer, audience and TTL (1m-24h) must be valid")
	}
	return &JWTManager{secret: []byte(secret), issuer: issuer, audience: audience, ttl: ttl}, nil
}

func (m *JWTManager) IssueAccessToken(id uint64) (*entities.AccessToken, error) {
	if id == 0 {
		return nil, fmt.Errorf("cannot issue token without a job seeker ID")
	}
	now := time.Now().UTC().Truncate(time.Second)
	expires := now.Add(m.ttl)
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	claims := accessClaims{
		Role: "job_seeker",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			Audience:  jwt.ClaimStrings{m.audience},
			Subject:   strconv.FormatUint(id, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expires),
			ID:        hex.EncodeToString(nonce),
		},
	}
	value, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return nil, err
	}
	return &entities.AccessToken{Value: value, ExpiresAt: expires}, nil
}

func (m *JWTManager) VerifyAccessToken(value string) (*entities.JobSeekerPrincipal, error) {
	if len(value) > 4096 {
		return nil, ports.ErrUnauthenticated
	}
	claims := &accessClaims{}
	token, err := jwt.ParseWithClaims(value, claims,
		func(_ *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(m.issuer),
		jwt.WithAudience(m.audience),
		jwt.WithIssuedAt(),
	)
	if err != nil || !token.Valid || claims.Role != "job_seeker" || claims.IssuedAt == nil || claims.ID == "" {
		return nil, ports.ErrUnauthenticated
	}
	id, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || id == 0 {
		return nil, ports.ErrUnauthenticated
	}
	return &entities.JobSeekerPrincipal{ID: id, TokenID: claims.ID, ExpiresAt: claims.ExpiresAt.Time}, nil
}
