package general

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/ports"
	"zedlancer/backend/internal/presentation/middleware"
)

type EmployerJobPostingController struct {
	service usecase.EmployerJobPostingService
}

func NewEmployerJobPostingController(service usecase.EmployerJobPostingService) *EmployerJobPostingController {
	return &EmployerJobPostingController{service: service}
}

type JobPostingCreatedResponse struct {
	ID     uint64 `json:"id" example:"42"`
	Status string `json:"status" example:"draft"`
}

// CreateWork godoc
// @Summary Create a work draft
// @ID createWorkPosting
// @Description Creates an employer-owned draft with a budget in whole toman and a delivery deadline after the application deadline. The route supplies opportunity_type=work. Dates are Gregorian YYYY-MM-DD. Requires a valid token for a phone-verified individual or legal employer with current admin approval. The response status is draft.
// @Tags Work postings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body saveWorkPostingRequest true "Work draft"
// @Success 201 {object} JobPostingCreatedResponse "Draft created"
// @Failure 400 {object} ErrorResponse "Invalid posting data"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/job-postings [post]
func (c *EmployerJobPostingController) CreateWork(ctx *gin.Context) {
	c.create(ctx, entities.WorkPosting)
}

// GetWork godoc
// @Summary Review your work draft
// @ID getWorkPosting
// @Description Returns your work draft and private attachment download URLs valid for five minutes. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Work postings
// @Produce json
// @Security BearerAuth
// @Param id path int true "Posting ID" minimum(1)
// @Success 200 {object} usecase.JobPostingReview{details=workPostingDetailsRequest}
// @Failure 400 {object} ErrorResponse "Invalid posting ID"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "Attachment storage unavailable"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Header 200 {string} Cache-Control "no-store"
// @Router /employer/job-postings/{id} [get]
func (c *EmployerJobPostingController) GetWork(ctx *gin.Context) {
	c.get(ctx, entities.WorkPosting)
}

// UpdateWork godoc
// @Summary Edit your work draft
// @ID updateWorkPosting
// @Description Replaces the work draft's title, description, and details. Existing attachments are updated through the attachment endpoint. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Work postings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Posting ID" minimum(1)
// @Param request body saveWorkPostingRequest true "Work draft"
// @Success 204 "Updated"
// @Failure 400 {object} ErrorResponse "Invalid posting ID or request data"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/job-postings/{id} [put]
func (c *EmployerJobPostingController) UpdateWork(ctx *gin.Context) {
	c.update(ctx, entities.WorkPosting)
}

// DeleteWork godoc
// @Summary Delete your work draft
// @ID deleteWorkPosting
// @Description Deletes your work draft and removes its attachments. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Work postings
// @Produce json
// @Security BearerAuth
// @Param id path int true "Posting ID" minimum(1)
// @Success 204 "Deleted"
// @Failure 400 {object} ErrorResponse "Invalid posting ID"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/job-postings/{id} [delete]
func (c *EmployerJobPostingController) DeleteWork(ctx *gin.Context) {
	c.delete(ctx, entities.WorkPosting)
}

// CreateInternship godoc
// @Summary Create an internship draft
// @ID createInternshipPosting
// @Description Creates an employer-owned draft for a paid or unpaid internship. The route supplies opportunity_type=internship. Dates are Gregorian YYYY-MM-DD. The description defaults to responsibilities. The example shows a paid internship. For an unpaid internship, set has_monthly_stipend to false and omit monthly_stipend_toman or set it to zero. Requires a valid token for a phone-verified individual or legal employer with current admin approval. The response status is draft.
// @Tags Internship postings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body saveInternshipPostingRequest true "Internship draft"
// @Success 201 {object} JobPostingCreatedResponse "Draft created"
// @Failure 400 {object} ErrorResponse "Invalid posting data"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/internship-postings [post]
func (c *EmployerJobPostingController) CreateInternship(ctx *gin.Context) {
	c.create(ctx, entities.InternshipPosting)
}

// GetInternship godoc
// @Summary Review your internship draft
// @ID getInternshipPosting
// @Description Returns your internship draft, including its scheduling and stipend terms. The attachments array is empty. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Internship postings
// @Produce json
// @Security BearerAuth
// @Param id path int true "Posting ID" minimum(1)
// @Success 200 {object} usecase.JobPostingReview{details=internshipPostingDetailsRequest}
// @Failure 400 {object} ErrorResponse "Invalid posting ID"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Header 200 {string} Cache-Control "no-store"
// @Router /employer/internship-postings/{id} [get]
func (c *EmployerJobPostingController) GetInternship(ctx *gin.Context) {
	c.get(ctx, entities.InternshipPosting)
}

// UpdateInternship godoc
// @Summary Edit your internship draft
// @ID updateInternshipPosting
// @Description Replaces the internship draft's title, description, and details. Dates are Gregorian YYYY-MM-DD. The description defaults to responsibilities. The example shows a paid internship. For an unpaid internship, set has_monthly_stipend to false and omit monthly_stipend_toman or set it to zero. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Internship postings
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Posting ID" minimum(1)
// @Param request body saveInternshipPostingRequest true "Internship draft"
// @Success 204 "Updated"
// @Failure 400 {object} ErrorResponse "Invalid posting ID or request data"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/internship-postings/{id} [put]
func (c *EmployerJobPostingController) UpdateInternship(ctx *gin.Context) {
	c.update(ctx, entities.InternshipPosting)
}

// DeleteInternship godoc
// @Summary Delete your internship draft
// @ID deleteInternshipPosting
// @Description Deletes your internship draft. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Internship postings
// @Produce json
// @Security BearerAuth
// @Param id path int true "Posting ID" minimum(1)
// @Success 204 "Deleted"
// @Failure 400 {object} ErrorResponse "Invalid posting ID"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/internship-postings/{id} [delete]
func (c *EmployerJobPostingController) DeleteInternship(ctx *gin.Context) {
	c.delete(ctx, entities.InternshipPosting)
}

func (c *EmployerJobPostingController) create(ctx *gin.Context, kind entities.JobPostingType) {
	token, ok := jobPostingToken(ctx)
	if !ok {
		return
	}
	input, err := bindJobPosting(ctx, kind)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	id, err := c.service.Create(ctx.Request.Context(), token, input)
	if err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, JobPostingCreatedResponse{ID: id, Status: "draft"})
}

func (c *EmployerJobPostingController) get(ctx *gin.Context, kind entities.JobPostingType) {
	token, id, ok := jobPostingAccess(ctx)
	if !ok {
		return
	}
	posting, err := c.service.Get(ctx.Request.Context(), token, id, kind)
	if err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, posting)
}

func (c *EmployerJobPostingController) update(ctx *gin.Context, kind entities.JobPostingType) {
	token, id, ok := jobPostingAccess(ctx)
	if !ok {
		return
	}
	input, err := bindJobPosting(ctx, kind)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	if err := c.service.Update(ctx.Request.Context(), token, id, input); err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func (c *EmployerJobPostingController) delete(ctx *gin.Context, kind entities.JobPostingType) {
	token, id, ok := jobPostingAccess(ctx)
	if !ok {
		return
	}
	if err := c.service.Delete(ctx.Request.Context(), token, id, kind); err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// ReplaceAttachments godoc
// @Summary Replace work advertisement attachments
// @ID replaceWorkPostingAttachments
// @Description Replaces all attachments on your work draft. Send up to five PDF, PNG, or JPEG files by repeating the attachments multipart field. Each file may be up to 10 MiB. Swagger UI provides one file picker. Use an HTTP client that can repeat multipart fields to upload multiple files together. An empty multipart form clears the attachments. Requires a valid token for a phone-verified individual or legal employer with current admin approval.
// @Tags Work postings
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param id path int true "Job posting ID" minimum(1)
// @Param attachments formData file false "Attachment; repeat the field for multiple files"
// @Success 204 "Attachments replaced"
// @Failure 400 {object} ErrorResponse "Invalid posting ID or request data"
// @Failure 401 {object} ErrorResponse "Valid employer token and verified phone required"
// @Failure 403 {object} ErrorResponse "Employer review is pending or rejected"
// @Failure 404 {object} ErrorResponse "Posting not found for this employer and posting type"
// @Failure 413 {object} ErrorResponse "Multipart request exceeds the upload limit"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "Attachment storage unavailable"
// @Header 401 {string} WWW-Authenticate "Bearer"
// @Router /employer/job-postings/{id}/attachments [put]
func (c *EmployerJobPostingController) ReplaceAttachments(ctx *gin.Context) {
	token, id, ok := jobPostingAccess(ctx)
	if !ok || !parseEmployerMultipart(ctx, usecase.MaxJobPostingAttachments*usecase.MaxJobPostingAttachmentBytes+128*1024) {
		return
	}
	defer ctx.Request.MultipartForm.RemoveAll()
	form := ctx.Request.MultipartForm
	headers := form.File["attachments"]
	if len(headers) > usecase.MaxJobPostingAttachments || len(form.Value) > 0 || len(form.File) > 1 || (len(form.File) > 0 && len(headers) == 0) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "use the attachments file field for up to five files"})
		return
	}
	var handles employerMultipartFiles
	defer handles.close()
	files := make([]ports.UploadFile, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "cannot open attachment"})
			return
		}
		handles.closers = append(handles.closers, file)
		files = append(files, ports.UploadFile{Name: header.Filename, Body: file})
	}
	if err := c.service.ReplaceAttachments(ctx.Request.Context(), token, id, files); err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

func jobPostingToken(ctx *gin.Context) (string, bool) {
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "Authorization header required: Bearer <access_token>"})
	}
	return token, ok
}

func jobPostingAccess(ctx *gin.Context) (string, uint64, bool) {
	token, ok := jobPostingToken(ctx)
	if !ok {
		return "", 0, false
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid job posting id"})
		return "", 0, false
	}
	return token, id, true
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
	case errors.Is(err, domainerrors.ErrInvalidUpload):
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
	case errors.Is(err, domainerrors.ErrStorageUnavailable):
		ctx.JSON(http.StatusServiceUnavailable, ErrorResponse{Message: err.Error()})
	case errors.Is(err, domainerrors.ErrJobPostingNotFound):
		ctx.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
	default:
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
	}
}
