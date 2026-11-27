package general

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/domain/entities"
	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/presentation/middleware"
)

type jobSeekerSkills struct {
	Skills []string `json:"skills" binding:"required,max=30" example:"طراحی پست,فتوشاپ"`
}

// ListJobSeekerOpportunities godoc
// @Summary Search the job seeker dashboard opportunities
// @ID listJobSeekerOpportunities
// @Description Searches published opportunities with an open application deadline. Search matches only the title. Choose a single tab. The for_me tab requires a skill overlap with the authenticated job seeker's saved profile and defaults to best_match. The score is the percentage of distinct posting skills present in the profile. The saved tab lists the current job seeker's saved opportunities.
// @Tags Job seekers
// @Security BearerAuth
// @Produce json
// @Param tab query string false "Single dashboard tab" Enums(all,for_me,work,internship,saved) default(all)
// @Param q query string false "Literal title substring" maxlength(255)
// @Param sort query string false "Sort order. Defaults to best_match for for_me and newest otherwise." Enums(newest,closest_deadline,highest_amount,best_match)
// @Param category query []string false "Exact posting category" collectionFormat(multi)
// @Param work_mode query []string false "Work mode" Enums(online,onsite,hybrid) collectionFormat(multi)
// @Param age_group query []string false "Age range overlaps 14–17 with parental approval or 18–23" Enums(under_18,18_to_23) collectionFormat(multi)
// @Param skill_level query []string false "Required skill level" Enums(beginner,intermediate,advanced) collectionFormat(multi)
// @Param min_amount_toman query int false "Inclusive minimum amount in whole toman" minimum(0) format(int64)
// @Param max_amount_toman query int false "Inclusive maximum amount in whole toman" minimum(0) format(int64)
// @Param duration query []string false "Duration buckets" Enums(under_3_days,up_to_week,up_to_month,over_month) collectionFormat(multi)
// @Param city query []string false "Employer's registered city in Persian" collectionFormat(multi)
// @Param limit query int false "Page size" minimum(1) maximum(100) default(20)
// @Param offset query int false "Number of matching results to skip" minimum(0) default(0)
// @Success 200 {object} entities.OpportunityPage
// @Failure 400,401,500 {object} ErrorResponse
// @Router /job-seekers/opportunities [get]
func (c *EmployerJobPostingController) ListJobSeekerOpportunities(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	tabs := ctx.QueryArray("tab")
	if len(tabs) > 1 || (len(tabs) == 1 && tabs[0] == "") || ctx.Request.URL.Query().Has("opportunity_type") {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "select a single dashboard tab with tab"})
		return
	}
	var query opportunityQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid opportunity query"})
		return
	}
	page, err := c.service.ListJobSeekerOpportunities(ctx.Request.Context(), seeker, entities.OpportunityTab(ctx.Query("tab")), entities.OpportunityFilters(query))
	ctx.Header("Cache-Control", "no-store")
	respondOpportunityPage(ctx, page, err)
}

// GetJobSeekerSkills godoc
// @Summary Read the current job seeker's skills
// @Tags Job seekers
// @Security BearerAuth
// @Produce json
// @Success 200 {object} jobSeekerSkills
// @Failure 401,500 {object} ErrorResponse
// @Router /job-seekers/me/skills [get]
func (c *EmployerJobPostingController) GetJobSeekerSkills(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, jobSeekerSkills{Skills: append([]string{}, seeker.Skills...)})
}

// UpdateJobSeekerSkills godoc
// @Summary Replace the current job seeker's skills
// @Description Saves up to 30 distinct skill names from the job seeker's skills profile. An empty array clears the list.
// @Tags Job seekers
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body jobSeekerSkills true "Skill names used for opportunity recommendations"
// @Success 204 "Skills replaced"
// @Failure 400,401,500 {object} ErrorResponse
// @Router /job-seekers/me/skills [put]
func (c *EmployerJobPostingController) UpdateJobSeekerSkills(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var request jobSeekerSkills
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "skills must be an array of skill names"})
		return
	}
	err := c.service.UpdateJobSeekerSkills(ctx.Request.Context(), seeker.ID, request.Skills)
	if errors.Is(err, domainerrors.ErrInvalidOpportunityFilters) {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	if err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}

// SaveOpportunity godoc
// @Summary Save a published opportunity to the current job seeker's dashboard
// @Tags Job seekers
// @Security BearerAuth
// @Produce json
// @Param id path int true "Opportunity ID" minimum(1)
// @Success 204 "Opportunity saved"
// @Failure 400,401,404,500 {object} ErrorResponse
// @Router /job-seekers/opportunities/{id}/saved [put]
func (c *EmployerJobPostingController) SaveOpportunity(ctx *gin.Context) {
	c.setOpportunitySaved(ctx, true)
}

// UnsaveOpportunity godoc
// @Summary Remove an opportunity from the current job seeker's saved list
// @Tags Job seekers
// @Security BearerAuth
// @Produce json
// @Param id path int true "Opportunity ID" minimum(1)
// @Success 204 "Opportunity removed"
// @Failure 400,401,500 {object} ErrorResponse
// @Router /job-seekers/opportunities/{id}/saved [delete]
func (c *EmployerJobPostingController) UnsaveOpportunity(ctx *gin.Context) {
	c.setOpportunitySaved(ctx, false)
}

func (c *EmployerJobPostingController) setOpportunitySaved(ctx *gin.Context, saved bool) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 63)
	if err != nil || id == 0 {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid opportunity id"})
		return
	}
	if saved {
		err = c.service.SaveOpportunity(ctx.Request.Context(), seeker.ID, id)
	} else {
		err = c.service.UnsaveOpportunity(ctx.Request.Context(), seeker.ID, id)
	}
	if err != nil {
		respondJobPostingError(ctx, err)
		return
	}
	ctx.Status(http.StatusNoContent)
}
