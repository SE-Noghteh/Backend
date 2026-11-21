package general

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/presentation/middleware"
)

type EmployerJobPostingController struct {
	service usecase.EmployerJobPostingService
}

func NewEmployerJobPostingController(service usecase.EmployerJobPostingService) *EmployerJobPostingController {
	return &EmployerJobPostingController{service: service}
}

type saveJobPostingRequest struct {
	Title       string                      `json:"title" binding:"required,max=255"`
	Description string                      `json:"description" binding:"required"`
	Details     *entities.JobPostingDetails `json:"details" binding:"required"`
}

type JobPostingCreatedResponse struct {
	ID     uint64 `json:"id"`
	Status string `json:"status" example:"draft"`
}

func (r saveJobPostingRequest) input() usecase.SaveJobPosting {
	return usecase.SaveJobPosting{Title: r.Title, Description: r.Description, Details: r.Details}
}

// Create godoc
// @Summary Create an employer job advertisement
// @Description Creates a draft for an approved individual or legal employer. Structured details are required for work and internship ads. Dates use Gregorian YYYY-MM-DD; money is whole toman. Applicants under 18 require parent approval. Creation does not charge a wallet or publish the ad.
// @Tags Employer job postings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body saveJobPostingRequest true "Job advertisement"
// @Success 201 {object} JobPostingCreatedResponse
// @Failure 400,401,403,500 {object} ErrorResponse
// @Router /employer/job-postings [post]
func (c *EmployerJobPostingController) Create(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 128*1024)
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "Authorization header required: Bearer <access_token>"})
		return
	}
	var request saveJobPostingRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "title and description are required"})
		return
	}
	id, err := c.service.Create(ctx.Request.Context(), token, request.input())
	if err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, JobPostingCreatedResponse{ID: id, Status: "draft"})
}

// Update godoc
// @Summary Edit your job advertisement
// @Description Replaces title, description and details. Requires an employer with approved verification. Returns 404 when the advertisement does not exist or belongs to another employer.
// @Tags Employer job postings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job posting ID" minimum(1)
// @Param request body saveJobPostingRequest true "Replacement advertisement fields"
// @Success 204 "Updated"
// @Failure 400,401,403,404,500 {object} ErrorResponse
// @Router /employer/job-postings/{id} [put]
func (c *EmployerJobPostingController) Update(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 128*1024)
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "Authorization header required: Bearer <access_token>"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid job posting id"})
		return
	}
	var request saveJobPostingRequest
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "title and description are required"})
		return
	}
	if err := c.service.Update(ctx.Request.Context(), token, id, request.input()); err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// Delete godoc
// @Summary Delete your job advertisement
// @Description Requires an employer with approved verification. Returns 404 when the advertisement does not exist or belongs to another employer.
// @Tags Employer job postings
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job posting ID" minimum(1)
// @Success 204 "Deleted"
// @Failure 400,401,403,404,500 {object} ErrorResponse
// @Router /employer/job-postings/{id} [delete]
func (c *EmployerJobPostingController) Delete(ctx *gin.Context) {
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "Authorization header required: Bearer <access_token>"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid job posting id"})
		return
	}
	if err := c.service.Delete(ctx.Request.Context(), token, id); err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func respondJobPostingError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrUnauthenticated):
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: err.Error()})
	case errors.Is(err, domainerrors.ErrEmployerNotApproved):
		ctx.JSON(http.StatusForbidden, ErrorResponse{Message: err.Error()})
	case errors.Is(err, domainerrors.ErrInvalidJobPosting):
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
	case errors.Is(err, domainerrors.ErrJobPostingNotFound):
		ctx.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
	default:
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
	}
}
