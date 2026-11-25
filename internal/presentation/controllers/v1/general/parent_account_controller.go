package general

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/presentation/middleware"
)

type ParentAccountController struct {
	service usecase.ParentAccountService
}

func NewParentAccountController(service usecase.ParentAccountService) *ParentAccountController {
	return &ParentAccountController{service: service}
}

type ParentAccountResponse struct {
	SignupIdentity
	valueobjects.AdminReview
	Relationship string `json:"relationship"`
}

type ParentLoginResponse struct {
	AccessTokenResponse
	Data ParentAccountResponse `json:"data"`
}

type ParentChildResponse struct {
	ID           uint64 `json:"id"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name"`
	SignupStatus string `json:"signup_status"`
}

type ParentConnectionResponse struct {
	JobSeekerID uint64    `json:"job_seeker_id"`
	FirstName   string    `json:"first_name"`
	LastName    string    `json:"last_name"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type ParentConnectionDecision struct {
	Approved *bool `json:"approved" binding:"required"`
}

func parentAccount(p *entities.Parent) ParentAccountResponse {
	return ParentAccountResponse{
		SignupIdentity: SignupIdentity{ID: p.ID, FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone, CreatedAt: p.CreatedAt},
		AdminReview:    p.AdminReview,
		Relationship:   p.Relationship,
	}
}

func (c *ParentAccountController) authenticated(ctx *gin.Context) (*entities.Parent, bool) {
	ctx.Header("Cache-Control", "no-store")
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		respondAPIError(ctx, domainerrors.ErrUnauthenticated)
		return nil, false
	}
	parent, err := c.service.Authenticate(ctx.Request.Context(), token)
	if err != nil {
		respondAPIError(ctx, err)
		return nil, false
	}
	return parent, true
}

// Login godoc
// @Summary Log in as a phone-verified parent
// @Tags Parents
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Parent phone and password"
// @Success 200 {object} ParentLoginResponse
// @Failure 400,401,429,500 {object} ErrorResponse
// @Router /parents/login [post]
func (c *ParentAccountController) Login(ctx *gin.Context) {
	var in LoginRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	parent, token, err := c.service.Login(ctx.Request.Context(), in.Phone, in.Password)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, ParentLoginResponse{
		AccessTokenResponse: AccessTokenResponse{AccessToken: token.Value, TokenType: "Bearer", ExpiresAt: token.ExpiresAt},
		Data:                parentAccount(parent),
	})
}

// Logout godoc
// @Summary Revoke the current parent session
// @Tags Parents
// @Security BearerAuth
// @Success 204
// @Failure 401,500 {object} ErrorResponse
// @Router /parents/logout [post]
func (c *ParentAccountController) Logout(ctx *gin.Context) {
	ctx.Header("Cache-Control", "no-store")
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		respondAPIError(ctx, domainerrors.ErrUnauthenticated)
		return
	}
	if err := c.service.Logout(ctx.Request.Context(), token); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Me godoc
// @Summary View the signed-in parent's profile and admin review status
// @Tags Parents
// @Security BearerAuth
// @Produce json
// @Success 200 {object} ParentAccountResponse
// @Failure 401,500 {object} ErrorResponse
// @Router /parents/me [get]
func (c *ParentAccountController) Me(ctx *gin.Context) {
	if parent, ok := c.authenticated(ctx); ok {
		ctx.JSON(http.StatusOK, parentAccount(parent))
	}
}

// Children godoc
// @Summary List the signed-in parent's connected children and signup status
// @Tags Parents
// @Security BearerAuth
// @Produce json
// @Success 200 {array} ParentChildResponse
// @Failure 401,500 {object} ErrorResponse
// @Router /parents/children [get]
func (c *ParentAccountController) Children(ctx *gin.Context) {
	parent, ok := c.authenticated(ctx)
	if !ok {
		return
	}
	rows, err := c.service.Children(ctx.Request.Context(), parent.ID)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	children := make([]ParentChildResponse, 0, len(rows))
	now := time.Now()
	for i := range rows {
		child := &rows[i]
		approval, err := child.ApprovalAt(now)
		if err != nil {
			respondAPIError(ctx, err)
			return
		}
		children = append(children, ParentChildResponse{ID: child.ID, FirstName: child.FirstName, LastName: child.LastName, SignupStatus: approval.SignupStatus})
	}
	ctx.JSON(http.StatusOK, children)
}

// ConnectionRequests godoc
// @Summary List the parent's pending connection requests
// @Tags Parents
// @Security BearerAuth
// @Produce json
// @Success 200 {array} ParentConnectionResponse
// @Failure 401,500 {object} ErrorResponse
// @Router /parents/connection-requests [get]
func (c *ParentAccountController) ConnectionRequests(ctx *gin.Context) {
	parent, ok := c.authenticated(ctx)
	if !ok {
		return
	}
	rows, err := c.service.ConnectionRequests(ctx.Request.Context(), parent.ID)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	requests := make([]ParentConnectionResponse, 0, len(rows))
	for _, row := range rows {
		requests = append(requests, ParentConnectionResponse{JobSeekerID: row.JobSeekerID, FirstName: row.FirstName, LastName: row.LastName, ExpiresAt: row.ExpiresAt})
	}
	ctx.JSON(http.StatusOK, requests)
}

// DecideConnection godoc
// @Summary Approve or reject a request to connect a job seeker
// @Tags Parents
// @Security BearerAuth
// @Accept json
// @Param job_seeker_id path int true "Job seeker ID"
// @Param request body ParentConnectionDecision true "Connection decision"
// @Success 204
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /parents/connection-requests/{job_seeker_id}/decision [post]
func (c *ParentAccountController) DecideConnection(ctx *gin.Context) {
	parent, ok := c.authenticated(ctx)
	if !ok {
		return
	}
	jobSeekerID, err := strconv.ParseUint(ctx.Param("job_seeker_id"), 10, 64)
	if err != nil || jobSeekerID == 0 {
		respondAPIError(ctx, domainerrors.ErrInvalidParent)
		return
	}
	var in ParentConnectionDecision
	if !bindParentJSON(ctx, &in) {
		return
	}
	if err := c.service.DecideConnection(ctx.Request.Context(), parent, jobSeekerID, *in.Approved); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

