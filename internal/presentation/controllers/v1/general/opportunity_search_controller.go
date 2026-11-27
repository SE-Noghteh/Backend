package general

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

type opportunityQuery struct {
	Title          string                         `form:"q"`
	Sort           entities.OpportunitySort       `form:"sort"`
	Categories     []string                       `form:"category"`
	WorkModes      []string                       `form:"work_mode"`
	AgeGroups      []entities.OpportunityAgeGroup `form:"age_group"`
	Types          []entities.JobPostingType      `form:"opportunity_type"`
	SkillLevels    []entities.PostingSkillLevel   `form:"skill_level"`
	MinAmountToman *int64                         `form:"min_amount_toman"`
	MaxAmountToman *int64                         `form:"max_amount_toman"`
	Durations      []entities.OpportunityDuration `form:"duration"`
	Cities         []string                       `form:"city"`
	Limit          int                            `form:"limit,default=20" binding:"min=1,max=100"`
	Offset         int                            `form:"offset" binding:"min=0"`
}

// GetOpportunity godoc
// @Summary Get a published work or internship opportunity
// @ID getOpportunity
// @Description Returns the public posting including description, skills, work or internship terms, and signed attachment links valid for five minutes. Employer verification reflects admin approval. A published posting with a past deadline returns accepting_applications=false. Published_at is included when recorded.
// @Tags Opportunities
// @Produce json
// @Param id path int true "Opportunity ID from the public listing" minimum(1)
// @Success 200 {object} usecase.OpportunityDetail
// @Failure 400 {object} ErrorResponse "Invalid opportunity ID"
// @Failure 404 {object} ErrorResponse "Published opportunity not found"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Failure 503 {object} ErrorResponse "Attachment storage unavailable"
// @Header 200 {string} Cache-Control "no-store"
// @Router /opportunities/{id} [get]
func (c *EmployerJobPostingController) GetOpportunity(ctx *gin.Context) {
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 63)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid opportunity id"})
		return
	}
	posting, err := c.service.GetOpportunity(ctx.Request.Context(), id)
	if err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, posting)
}

// ListOpportunities godoc
// @Summary Search published work and internship opportunities
// @ID listOpportunities
// @Description Lists published postings with an application deadline of today or later in Iran. Search matches only the title. Multiple selections within a filter match any selected value. Different filters are combined. Work amounts use the project budget and internship amounts use the monthly stipend. Unpaid internships have zero amount.
// @Tags Opportunities
// @Produce json
// @Param q query string false "Literal title substring" maxlength(255)
// @Param sort query string false "Sort by creation date, application deadline or amount" Enums(newest,closest_deadline,highest_amount) default(newest)
// @Param category query []string false "Exact posting category; repeat for multiple selections" collectionFormat(multi)
// @Param work_mode query []string false "Work mode" Enums(online,onsite,hybrid) collectionFormat(multi)
// @Param age_group query []string false "Age range overlaps 14–17 with parental approval or 18–23" Enums(under_18,18_to_23) collectionFormat(multi)
// @Param opportunity_type query []string false "Project or internship" Enums(work,internship) collectionFormat(multi)
// @Param skill_level query []string false "Required skill level" Enums(beginner,intermediate,advanced) collectionFormat(multi)
// @Param min_amount_toman query int false "Inclusive minimum amount in whole toman" minimum(0) format(int64)
// @Param max_amount_toman query int false "Inclusive maximum amount in whole toman" minimum(0) format(int64)
// @Param duration query []string false "Duration buckets in days: 1–2, 3–7, 8–30, over 30. An internship month counts as 30 days." Enums(under_3_days,up_to_week,up_to_month,over_month) collectionFormat(multi)
// @Param city query []string false "Employer's registered city in Persian; repeat for multiple selections" collectionFormat(multi)
// @Param limit query int false "Page size" minimum(1) maximum(100) default(20)
// @Param offset query int false "Number of matching results to skip" minimum(0) default(0)
// @Success 200 {object} entities.OpportunityPage
// @Failure 400 {object} ErrorResponse "Invalid query or filter values"
// @Failure 500 {object} ErrorResponse "Internal server error"
// @Router /opportunities [get]
func (c *EmployerJobPostingController) ListOpportunities(ctx *gin.Context) {
	var query opportunityQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid opportunity query"})
		return
	}
	page, err := c.service.ListOpportunities(ctx.Request.Context(), entities.OpportunityFilters(query))
	respondOpportunityPage(ctx, page, err)
}

func respondOpportunityPage(ctx *gin.Context, page *entities.OpportunityPage, err error) {
	switch {
	case errors.Is(err, domainerrors.ErrInvalidOpportunityFilters):
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
	case err != nil:
		ctx.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
	default:
		ctx.JSON(http.StatusOK, page)
	}
}
