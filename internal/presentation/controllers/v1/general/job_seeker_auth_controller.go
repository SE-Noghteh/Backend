package general

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/presentation/middleware"
)

type JobSeekerAuthController struct{ service usecase.JobSeekerAuthService }

func NewJobSeekerAuthController(service usecase.JobSeekerAuthService) *JobSeekerAuthController {
	return &JobSeekerAuthController{service: service}
}

// Login godoc
// @Summary Login with phone and password
// @Description Requires a verified account. Returns an expiring JWT. Swagger automatically authorizes subsequent requests after successful login. Other clients must send Authorization: Bearer <access_token>. Five failed attempts per phone within 15 minutes block further login attempts until the window ends.
// @Tags Job seeker authentication
// @Accept json
// @Produce json
// @Param request body usecase.LoginJobSeeker true "Phone and password"
// @Success 200 {object} LoginResponse
// @Failure 400 {object} SignupErrorResponse
// @Failure 401 {object} SignupErrorResponse
// @Failure 429 {object} SignupErrorResponse
// @Failure 500 {object} SignupErrorResponse
// @Router /job-seekers/login [post]
func (c *JobSeekerAuthController) Login(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in usecase.LoginJobSeeker
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, SignupErrorResponse{Message: "invalid JSON body"})
		return
	}
	result, err := c.service.Login(ctx.Request.Context(), in)
	if err != nil {
		signupError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")
	ctx.JSON(http.StatusOK, LoginResponse{AccessToken: result.AccessToken.Value, TokenType: "Bearer", ExpiresAt: result.AccessToken.ExpiresAt, Data: identityFromJobSeeker(result.JobSeeker)})
}

// Me godoc
// @Summary Get the logged-in job seeker
// @Description Returns the identity of the verified account identified by the JWT.
// @Tags Job seeker authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SignupResponse
// @Failure 401 {object} SignupErrorResponse
// @Failure 500 {object} SignupErrorResponse
// @Router /job-seekers/me [get]
func (c *JobSeekerAuthController) Me(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, SignupErrorResponse{Message: "authentication required"})
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, SignupResponse{Data: identityFromJobSeeker(seeker)})
}

// Logout godoc
// @Summary Logout the current access token
// @Description Permanently revokes this token until its expiration. Other login sessions stay active. Repeating logout with the same unexpired token returns 204. Swagger clears its authorization after successful logout.
// @Tags Job seeker authentication
// @Security BearerAuth
// @Success 204 "Logged out"
// @Failure 401 {object} SignupErrorResponse
// @Failure 500 {object} SignupErrorResponse
// @Router /job-seekers/logout [post]
func (c *JobSeekerAuthController) Logout(ctx *gin.Context) {
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, SignupErrorResponse{Message: "Authorization header required: Bearer <access_token>"})
		return
	}
	if err := c.service.Logout(ctx.Request.Context(), token); err != nil {
		signupError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Status(http.StatusNoContent)
}
