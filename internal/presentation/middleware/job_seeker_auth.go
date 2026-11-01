package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

const jobSeekerContextKey = "authenticated_job_seeker"

type JobSeekerAuth struct{ service usecase.JobSeekerAuthService }

func NewJobSeekerAuth(service usecase.JobSeekerAuthService) *JobSeekerAuth {
	return &JobSeekerAuth{service: service}
}

func (m *JobSeekerAuth) Require(ctx *gin.Context) {
	token, ok := BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization header required: Bearer <access_token>"})
		return
	}
	seeker, err := m.service.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, ports.ErrUnauthenticated) {
		unauthorized(ctx)
		return
	}
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		return
	}
	ctx.Set(jobSeekerContextKey, seeker)
	ctx.Next()
}

func BearerToken(ctx *gin.Context) (string, bool) {
	parts := strings.Fields(ctx.GetHeader("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return "", false
	}
	return parts[1], true
}

func unauthorized(ctx *gin.Context) {
	ctx.Header("WWW-Authenticate", "Bearer")
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": ports.ErrUnauthenticated.Error()})
}

func CurrentJobSeeker(ctx *gin.Context) (*entities.JobSeeker, bool) {
	value, exists := ctx.Get(jobSeekerContextKey)
	if !exists {
		return nil, false
	}
	seeker, ok := value.(*entities.JobSeeker)
	return seeker, ok && seeker != nil
}
