package middleware

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"zedlancer/backend/internal/application/usecase"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

const adminContextKey = "authenticated_admin"

type AdminAuth struct{ service usecase.AdminService }

func NewAdminAuth(service usecase.AdminService) *AdminAuth { return &AdminAuth{service: service} }
func (m *AdminAuth) Require(ctx *gin.Context) {
	token, ok := BearerToken(ctx)
	if !ok {
		unauthorized(ctx)
		return
	}
	username, err := m.service.Authenticate(ctx.Request.Context(), token)
	if errors.Is(err, domainerrors.ErrUnauthenticated) {
		unauthorized(ctx)
		return
	}
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "internal server error"})
		return
	}
	ctx.Set(adminContextKey, username)
	ctx.Header("Cache-Control", "no-store")
	ctx.Next()
}
func CurrentAdmin(ctx *gin.Context) string {
	value, _ := ctx.Get(adminContextKey)
	username, _ := value.(string)
	return username
}
