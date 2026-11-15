package general

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/presentation/middleware"
)

type AdminController struct{ service usecase.AdminService }

func NewAdminController(service usecase.AdminService) *AdminController {
	return &AdminController{service: service}
}

type AdminLoginRequest struct {
	Username string `json:"username" binding:"required" example:"admin"`
	Password string `json:"password" binding:"required" example:"your-admin-password"`
}
type AdminLoginResponse struct {
	AccessTokenResponse
	Username string `json:"username"`
}
type AdminReviewRequest struct {
	Approved *bool  `json:"approved" binding:"required"`
	Message  string `json:"message" example:"Please submit a clearer video"`
}
type AdminJobSeekerResponse struct {
	JobSeekerIdentity
	NationalCode         string           `json:"national_code"`
	City                 enums.City       `json:"city"`
	Address              string           `json:"address"`
	BirthDate            string           `json:"birth_date"`
	Gender               enums.Gender     `json:"gender"`
	Interests            []enums.Interest `json:"interests"`
	HasVerificationVideo bool             `json:"has_verification_video"`
}
type AdminParentResponse struct {
	SignupIdentity
	valueobjects.AdminReview
	Relationship        string `json:"relationship"`
	HasIdentityDocument bool   `json:"has_identity_document"`
	HasConsentVideo     bool   `json:"has_consent_video"`
}
type AdminJobSeekersResponse struct {
	Data  []AdminJobSeekerResponse `json:"data"`
	Total int64                    `json:"total"`
}
type AdminParentsResponse struct {
	Data  []AdminParentResponse `json:"data"`
	Total int64                 `json:"total"`
}
type AdminJobSeekerReviewResponse struct {
	Message string                 `json:"message"`
	Data    AdminJobSeekerResponse `json:"data"`
}
type AdminParentReviewResponse struct {
	Message string              `json:"message"`
	Data    AdminParentResponse `json:"data"`
}

func adminJobSeeker(p *entities.JobSeeker) AdminJobSeekerResponse {
	return AdminJobSeekerResponse{JobSeekerIdentity: identityFromJobSeeker(p), NationalCode: p.NationalCode, City: p.City, Address: p.Address, BirthDate: p.BirthDate.String(), Gender: p.Gender, Interests: p.Interests, HasVerificationVideo: p.VerificationVideo != nil}
}
func adminParent(p *entities.Parent) AdminParentResponse {
	review := p.AdminReview
	if review.AdminApprovalStatus == "" {
		review.AdminApprovalStatus = enums.AdminReviewPending
	}
	return AdminParentResponse{SignupIdentity: SignupIdentity{ID: p.ID, FirstName: p.FirstName, LastName: p.LastName, Phone: p.Phone, CreatedAt: p.CreatedAt}, AdminReview: review, Relationship: p.Relationship, HasIdentityDocument: p.IdentityDocument != nil, HasConsentVideo: p.ConsentVideo != nil}
}
func adminAccountID(ctx *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 64)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid account id"})
		return 0, false
	}
	return id, true
}
func adminFilter(ctx *gin.Context) (usecase.AdminListFilter, bool) {
	var query struct {
		Status string `form:"status"`
		Limit  int    `form:"limit"`
		Offset int    `form:"offset"`
	}
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid query parameters"})
		return usecase.AdminListFilter{}, false
	}
	return usecase.AdminListFilter{Status: query.Status, Limit: query.Limit, Offset: query.Offset}, true
}

// Login godoc
// @Summary Admin login with username and password
// @Description No signup or OTP. Admin credentials are provisioned through SQL. Passwords must be bcrypt hashes in the database. Five failed login attempts per username within 15 minutes trigger a temporary lockout. Returns an admin-only JWT and automatically authorizes Swagger.
// @Tags Admin
// @Accept json
// @Produce json
// @Param request body AdminLoginRequest true "Admin credentials"
// @Success 200 {object} AdminLoginResponse
// @Failure 400,401,429,500 {object} ErrorResponse
// @Router /admin/login [post]
func (c *AdminController) Login(ctx *gin.Context) {
	var in AdminLoginRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	result, err := c.service.Login(ctx.Request.Context(), in.Username, in.Password)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, AdminLoginResponse{Username: result.Username, AccessTokenResponse: AccessTokenResponse{AccessToken: result.AccessToken.Value, TokenType: "Bearer", ExpiresAt: result.AccessToken.ExpiresAt}})
}

// JobSeekers godoc
// @Summary List job seekers ready for admin review
// @Description Lists minors with a connected parent and adults aged 18 or above, including adults who have not used the optional parent stage. Defaults to pending overall acceptance, excluding rejected job seeker reviews but including approved minors awaiting parent admin approval. Minors are approved only when all three approval flags are true; adults need only admin_approved. Pending users can still log in.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param status query string false "Overall approval (rejected selects job seeker admin rejection)" Enums(pending,approved,rejected,all) default(pending)
// @Param limit query int false "Page size (1-100)" default(20)
// @Param offset query int false "Nonnegative offset" default(0)
// @Success 200 {object} AdminJobSeekersResponse
// @Failure 400,401,500 {object} ErrorResponse
// @Router /admin/job-seekers [get]
func (c *AdminController) JobSeekers(ctx *gin.Context) {
	filter, ok := adminFilter(ctx)
	if !ok {
		return
	}
	rows, total, err := c.service.JobSeekers(ctx.Request.Context(), filter)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	data := make([]AdminJobSeekerResponse, 0, len(rows))
	for _, row := range rows {
		data = append(data, adminJobSeeker(row))
	}
	ctx.JSON(http.StatusOK, AdminJobSeekersResponse{Data: data, Total: total})
}

// JobSeeker godoc
// @Summary Get a job seeker's profile for review
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "Job seeker ID"
// @Success 200 {object} AdminJobSeekerResponse
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /admin/job-seekers/{id} [get]
func (c *AdminController) JobSeeker(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	row, err := c.service.JobSeeker(ctx.Request.Context(), id)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, adminJobSeeker(row))
}

// JobSeekerVideo godoc
// @Summary Download a job seeker's verification video
// @Description Returns a private signed download URL valid for five minutes.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "Job seeker ID"
// @Success 200 {object} DocumentURLResponse
// @Failure 400,401,404,500,503 {object} ErrorResponse
// @Router /admin/job-seekers/{id}/documents/video [get]
func (c *AdminController) JobSeekerVideo(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	url, err := c.service.JobSeekerVideoURL(ctx.Request.Context(), id)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, DocumentURLResponse{URL: url, ExpiresIn: 300})
}

// ReviewJobSeeker godoc
// @Summary Approve or reject a job seeker
// @Description Minors require a connected parent before review and remain pending until that parent also receives admin approval. Adults aged 18 and above need only their own admin approval; the optional parent stage is not required. Submit approved=true or false; message is optional. The decision and message are visible in job seeker login and /me responses. Rejection without a message uses a default message.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Job seeker ID"
// @Param request body AdminReviewRequest true "Approval decision and optional message"
// @Success 200 {object} AdminJobSeekerReviewResponse
// @Failure 400,401,404,409,500 {object} ErrorResponse
// @Router /admin/job-seekers/{id}/review [post]
func (c *AdminController) ReviewJobSeeker(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	var in AdminReviewRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	row, err := c.service.ReviewJobSeeker(ctx.Request.Context(), id, usecase.AdminReviewDecision{Approved: *in.Approved, Message: in.Message, Reviewer: middleware.CurrentAdmin(ctx)})
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	message := "Job seeker approved successfully"
	if !*in.Approved {
		message = "Job seeker was not approved"
	}
	ctx.JSON(http.StatusOK, AdminJobSeekerReviewResponse{Message: message, Data: adminJobSeeker(row)})
}

// Parents godoc
// @Summary List parents awaiting or having received admin review
// @Description Defaults to pending parent reviews. Rejected parents are returned only by status=rejected or status=all.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param status query string false "Review status" Enums(pending,approved,rejected,all) default(pending)
// @Param limit query int false "Page size (1-100)" default(20)
// @Param offset query int false "Nonnegative offset" default(0)
// @Success 200 {object} AdminParentsResponse
// @Failure 400,401,500 {object} ErrorResponse
// @Router /admin/parents [get]
func (c *AdminController) Parents(ctx *gin.Context) {
	filter, ok := adminFilter(ctx)
	if !ok {
		return
	}
	rows, total, err := c.service.Parents(ctx.Request.Context(), filter)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	data := make([]AdminParentResponse, 0, len(rows))
	for _, row := range rows {
		data = append(data, adminParent(row))
	}
	ctx.JSON(http.StatusOK, AdminParentsResponse{Data: data, Total: total})
}

// Parent godoc
// @Summary Get a parent profile for review
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "Parent ID"
// @Success 200 {object} AdminParentResponse
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /admin/parents/{id} [get]
func (c *AdminController) Parent(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	row, err := c.service.Parent(ctx.Request.Context(), id)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, adminParent(row))
}

// ParentDocument godoc
// @Summary Download a parent's identity image or consent video
// @Description Returns a private signed download URL valid for five minutes.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "Parent ID"
// @Param kind path string true "Document kind" Enums(identity,consent-video)
// @Success 200 {object} DocumentURLResponse
// @Failure 400,401,404,500,503 {object} ErrorResponse
// @Router /admin/parents/{id}/documents/{kind} [get]
func (c *AdminController) ParentDocument(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	url, err := c.service.ParentDocumentURL(ctx.Request.Context(), id, ctx.Param("kind"))
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, DocumentURLResponse{URL: url, ExpiresIn: 300})
}

// ReviewParent godoc
// @Summary Approve or reject a parent
// @Description Submit approved=true or false with an optional message. A parent's admin review does not overwrite their children's own admin review decisions, but controls overall acceptance for all linked minors. Rejecting a parent returns those minors to pending; adults do not depend on parent approval.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Parent ID"
// @Param request body AdminReviewRequest true "Approval decision and optional message"
// @Success 200 {object} AdminParentReviewResponse
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /admin/parents/{id}/review [post]
func (c *AdminController) ReviewParent(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	var in AdminReviewRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	row, err := c.service.ReviewParent(ctx.Request.Context(), id, usecase.AdminReviewDecision{Approved: *in.Approved, Message: in.Message, Reviewer: middleware.CurrentAdmin(ctx)})
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	message := "Parent approved successfully"
	if !*in.Approved {
		message = "Parent was not approved"
	}
	ctx.JSON(http.StatusOK, AdminParentReviewResponse{Message: message, Data: adminParent(row)})
}
