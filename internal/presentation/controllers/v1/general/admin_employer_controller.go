package general

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/presentation/middleware"
)

type AdminEmployerSummary struct {
	ID           uint64                     `json:"id"`
	EmployerType enums.EmployerType         `json:"employer_type"`
	Name         string                     `json:"name"`
	Phone        string                     `json:"phone"`
	Province     enums.Province             `json:"province" swaggertype:"string"`
	City         enums.City                 `json:"city" swaggertype:"string"`
	ReviewStatus enums.EmployerReviewStatus `json:"review_status"`
	CreatedAt    time.Time                  `json:"created_at"`
}

type AdminEmployersResponse struct {
	Data  []AdminEmployerSummary `json:"data"`
	Total int64                  `json:"total"`
}

type AdminIndividualEmployerProfile struct {
	IndividualEmployerIdentity
	FatherName       string         `json:"father_name"`
	NationalCode     string         `json:"national_code"`
	Gender           enums.Gender   `json:"gender"`
	BirthDate        string         `json:"birth_date"`
	Province         enums.Province `json:"province" swaggertype:"string"`
	City             enums.City     `json:"city" swaggertype:"string"`
	Address          string         `json:"address"`
	PostalCode       string         `json:"postal_code"`
	BusinessName     *string        `json:"business_name"`
	BusinessActivity *string        `json:"business_activity"`
	Website          *string        `json:"website"`
	HasLegalGuardian bool           `json:"has_legal_guardian"`
	Email            *string        `json:"email"`
}

type AdminEmployerResponse struct {
	ID                 uint64                          `json:"id"`
	EmployerType       enums.EmployerType              `json:"employer_type"`
	ReviewStatus       enums.EmployerReviewStatus      `json:"review_status"`
	AvailableDocuments []string                        `json:"available_documents"`
	Individual         *AdminIndividualEmployerProfile `json:"individual,omitempty"`
	Legal              *LegalEmployerIdentity          `json:"legal,omitempty"`
}

type AdminEmployerReviewResponse struct {
	Message string                `json:"message"`
	Data    AdminEmployerResponse `json:"data"`
}

type employerDocument struct {
	kind string
	file *valueobjects.StoredObject
}

func availableEmployerDocuments(documents ...employerDocument) []string {
	kinds := make([]string, 0, len(documents))
	for _, doc := range documents {
		if doc.file != nil && doc.file.Key != "" {
			kinds = append(kinds, doc.kind)
		}
	}
	return kinds
}

func adminEmployer(profile *entities.EmployerReviewProfile) AdminEmployerResponse {
	if row := profile.Individual; row != nil {
		return AdminEmployerResponse{
			ID: row.Employer.ID, EmployerType: row.Employer.Type, ReviewStatus: row.ReviewStatus,
			AvailableDocuments: availableEmployerDocuments(
				employerDocument{"identity", &row.IdentityDocument},
				employerDocument{"video", &row.VerificationVideo},
				employerDocument{"business-license", row.BusinessLicense},
			),
			Individual: &AdminIndividualEmployerProfile{
				IndividualEmployerIdentity: identityFromIndividualEmployer(row),
				FatherName:                 row.FatherName, NationalCode: row.NationalCode, Gender: row.Gender, BirthDate: row.BirthDate.String(),
				Province: row.Province, City: row.City, Address: row.Address, PostalCode: row.PostalCode,
				BusinessName: row.BusinessName, BusinessActivity: row.BusinessActivity, Website: row.Website,
				HasLegalGuardian: row.HasLegalGuardian, Email: row.Email,
			},
		}
	}
	row := profile.Legal
	detail := identityFromLegalEmployer(row)
	return AdminEmployerResponse{
		ID: row.Employer.ID, EmployerType: row.Employer.Type, ReviewStatus: row.ReviewStatus,
		AvailableDocuments: availableEmployerDocuments(
			employerDocument{"establishment", &row.EstablishmentDocument},
			employerDocument{"identity", &row.IdentityDocument},
			employerDocument{"video", &row.VerificationVideo},
			employerDocument{"activity-license", row.ActivityLicense},
			employerDocument{"authorization", row.AuthorizationDocument},
		),
		Legal: &detail,
	}
}

// Employers godoc
// @Summary List employers for admin review
// @Description Lists phone-verified employers oldest first. Defaults to pending_review across both employer types.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param type query string false "Employer type" Enums(individual,legal,all) default(all)
// @Param status query string false "Review status" Enums(pending_review,approved,rejected,all) default(pending_review)
// @Param limit query int false "Page size (1-100)" default(20)
// @Param offset query int false "Nonnegative offset" default(0)
// @Success 200 {object} AdminEmployersResponse
// @Failure 400,401,500 {object} ErrorResponse
// @Router /admin/employers [get]
func (c *AdminController) Employers(ctx *gin.Context) {
	filter, ok := adminFilter(ctx)
	if !ok {
		return
	}
	rows, total, err := c.service.Employers(ctx.Request.Context(), usecase.AdminEmployerListFilter{
		AdminListFilter: filter, Type: enums.EmployerType(ctx.Query("type")),
	})
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	data := make([]AdminEmployerSummary, 0, len(rows))
	for _, row := range rows {
		data = append(data, AdminEmployerSummary(row))
	}
	ctx.JSON(http.StatusOK, AdminEmployersResponse{Data: data, Total: total})
}

// Employer godoc
// @Summary Get an employer profile and available documents
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "Employer ID"
// @Success 200 {object} AdminEmployerResponse
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /admin/employers/{id} [get]
func (c *AdminController) Employer(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	profile, err := c.service.Employer(ctx.Request.Context(), id)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, adminEmployer(profile))
}

// EmployerDocument godoc
// @Summary Get a private employer document download URL
// @Description The link is valid for five minutes. Use an available_documents value from the employer profile.
// @Tags Admin
// @Security BearerAuth
// @Produce json
// @Param id path int true "Employer ID"
// @Param kind path string true "Document kind" Enums(identity,video,business-license,establishment,activity-license,authorization)
// @Success 200 {object} DocumentURLResponse
// @Failure 400,401,404,500,503 {object} ErrorResponse
// @Router /admin/employers/{id}/documents/{kind} [get]
func (c *AdminController) EmployerDocument(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	url, err := c.service.EmployerDocumentURL(ctx.Request.Context(), id, ctx.Param("kind"))
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, DocumentURLResponse{URL: url, ExpiresIn: 300})
}

// ReviewEmployer godoc
// @Summary Approve or reject an individual or legal employer
// @Description Records the decision, optional message, and review time. Approved employers can manage job postings. Employers can see their current review status and message in /me.
// @Tags Admin
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "Employer ID"
// @Param request body AdminReviewRequest true "Approval decision and optional message"
// @Success 200 {object} AdminEmployerReviewResponse
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /admin/employers/{id}/review [post]
func (c *AdminController) ReviewEmployer(ctx *gin.Context) {
	id, ok := adminAccountID(ctx)
	if !ok {
		return
	}
	var in AdminReviewRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	profile, err := c.service.ReviewEmployer(ctx.Request.Context(), id, usecase.AdminReviewDecision{
		Approved: *in.Approved, Message: in.Message, Reviewer: middleware.CurrentAdmin(ctx),
	})
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	message := "Employer approved successfully"
	if !*in.Approved {
		message = "Employer was not approved"
	}
	ctx.JSON(http.StatusOK, AdminEmployerReviewResponse{Message: message, Data: adminEmployer(profile)})
}
