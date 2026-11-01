package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

type authStub struct{ err error }

func (s authStub) Logout(context.Context, string) error { return s.err }

func (s authStub) Login(context.Context, usecase.LoginJobSeeker) (*usecase.JobSeekerLoginResult, error) {
	return nil, nil
}
func (s authStub) Authenticate(_ context.Context, token string) (*entities.JobSeeker, error) {
	if s.err != nil {
		return nil, s.err
	}
	if token != "valid-token" {
		return nil, ports.ErrUnauthenticated
	}
	return &entities.JobSeeker{ID: 42}, nil
}

func TestJobSeekerAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, tt := range []struct {
		name, header string
		err          error
		status       int
	}{
		{name: "missing", status: 401},
		{name: "basic", header: "Basic valid-token", status: 401},
		{name: "missing token", header: "Bearer", status: 401},
		{name: "extra fields", header: "Bearer valid-token extra", status: 401},
		{name: "invalid token", header: "Bearer wrong-token", status: 401},
		{name: "valid", header: "Bearer valid-token", status: 200},
		{name: "case insensitive scheme", header: "bearer valid-token", status: 200},
		{name: "database failure", header: "Bearer valid-token", err: errors.New("private database error"), status: 500},
	} {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			called := false
			auth := NewJobSeekerAuth(authStub{err: tt.err})
			router.GET("/me", auth.Require, func(ctx *gin.Context) {
				called = true
				seeker, ok := CurrentJobSeeker(ctx)
				if !ok || seeker.ID != 42 {
					t.Fatal("authenticated identity missing")
				}
				ctx.Status(http.StatusOK)
			})
			request := httptest.NewRequest(http.MethodGet, "/me", nil)
			request.Header.Set("Authorization", tt.header)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != tt.status || called != (tt.status == 200) {
				t.Fatalf("status=%d protected handler called=%v", response.Code, called)
			}
			if tt.status == 401 && response.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Fatal("missing authentication challenge")
			}
		})
	}
}
