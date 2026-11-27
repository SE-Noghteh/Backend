package general

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
)

type postingTermsRequest struct {
	Category string `json:"category" binding:"required" minLength:"1" maxLength:"200" example:"طراحی گرافیک"`
	// Optional distinct skill names. At most 30 names, each at most 100 characters.
	Skills     []string                   `json:"skills" binding:"max=30,unique" example:"طراحی گرافیک,تایپوگرافی"`
	SkillLevel entities.PostingSkillLevel `json:"skill_level,omitempty" swaggertype:"string" enums:"beginner,intermediate,advanced" example:"beginner"`
	// Gregorian application deadline in YYYY-MM-DD format.
	ApplyDeadline string `json:"apply_deadline" binding:"required" format:"date" example:"2027-01-01"`
	WorkMode      string `json:"work_mode" binding:"required" enums:"online,onsite,hybrid" example:"online"`
	MinimumAge    int    `json:"minimum_age" binding:"required" minimum:"14" maximum:"100" example:"14"`
	// Must be at least minimum_age.
	MaximumAge int `json:"maximum_age" binding:"required" minimum:"14" maximum:"100" example:"18"`
	// Set both under-18 flags to true when minimum_age is below 18 and false otherwise.
	AcceptUnder18          bool `json:"accept_under_18" example:"true"`
	RequiresParentApproval bool `json:"requires_parent_approval" example:"true"`
}

func (r postingTermsRequest) details(kind entities.JobPostingType) *entities.JobPostingDetails {
	return &entities.JobPostingDetails{
		OpportunityType: kind, Category: r.Category, Skills: r.Skills, SkillLevel: r.SkillLevel, ApplyDeadline: r.ApplyDeadline,
		WorkMode: r.WorkMode, MinimumAge: r.MinimumAge, MaximumAge: r.MaximumAge,
		AcceptUnder18: r.AcceptUnder18, RequiresParentApproval: r.RequiresParentApproval,
	}
}

type workPostingDetailsRequest struct {
	postingTermsRequest
	OpportunityType *entities.JobPostingType `json:"opportunity_type,omitempty" swaggertype:"string" enums:"work" example:"work"`
	// Gregorian delivery deadline. Must be after apply_deadline.
	DeliveryDeadline string `json:"delivery_deadline" binding:"required" format:"date" example:"2027-02-01"`
	// Estimated time to complete the project, in days. Omit to use delivery_deadline minus apply_deadline.
	DurationDays    int      `json:"duration_days,omitempty" minimum:"1" example:"7"`
	Deliverables    []string `json:"deliverables,omitempty" example:"۱۰ پست,۱۰ استوری"`
	DeliveryFormats []string `json:"delivery_formats,omitempty" example:"PNG,PSD"`
	// Collaboration budget in whole toman.
	BudgetToman int64 `json:"budget_toman" binding:"required" minimum:"1" example:"2400000"`
}

type internshipPostingDetailsRequest struct {
	postingTermsRequest
	OpportunityType *entities.JobPostingType  `json:"opportunity_type,omitempty" swaggertype:"string" enums:"internship" example:"internship"`
	Internship      *entities.InternshipTerms `json:"internship" binding:"required"`
}

type saveWorkPostingRequest struct {
	Title string `json:"title" binding:"required,max=255" example:"طراحی پست‌های پاییز"`
	// Required work description. Maximum 50,000 bytes.
	Description string                     `json:"description" binding:"required" example:"طراحی ده پست اینستاگرامی با حال‌وهوای پاییزی و هماهنگ با هویت برند."`
	Details     *workPostingDetailsRequest `json:"details" binding:"required"`
}

type saveInternshipPostingRequest struct {
	Title string `json:"title" binding:"required,max=255" example:"کارآموزی تولید محتوا"`
	// Defaults to internship responsibilities when omitted or blank. Maximum 50,000 bytes.
	Description string                           `json:"description,omitempty" example:""`
	Details     *internshipPostingDetailsRequest `json:"details" binding:"required"`
}

func bindJobPosting(ctx *gin.Context, kind entities.JobPostingType) (usecase.SaveJobPosting, error) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 128*1024)
	decoder := json.NewDecoder(ctx.Request.Body)
	decoder.DisallowUnknownFields()
	if kind == entities.WorkPosting {
		var request saveWorkPostingRequest
		if err := decodeJobPosting(decoder, &request); err != nil {
			return usecase.SaveJobPosting{}, err
		}
		if request.Details == nil {
			return usecase.SaveJobPosting{}, errors.New("details are required")
		}
		if request.Details.OpportunityType != nil && *request.Details.OpportunityType != kind {
			return usecase.SaveJobPosting{}, errors.New("opportunity_type must be work for this route")
		}
		details := request.Details.details(kind)
		details.BudgetToman = request.Details.BudgetToman
		details.DeliveryDeadline = request.Details.DeliveryDeadline
		details.DurationDays = request.Details.DurationDays
		details.Deliverables = request.Details.Deliverables
		details.DeliveryFormats = request.Details.DeliveryFormats
		return usecase.SaveJobPosting{Title: request.Title, Description: request.Description, Details: details}, nil
	}

	var request saveInternshipPostingRequest
	if err := decodeJobPosting(decoder, &request); err != nil {
		return usecase.SaveJobPosting{}, err
	}
	if request.Details == nil {
		return usecase.SaveJobPosting{}, errors.New("details are required")
	}
	if request.Details.OpportunityType != nil && *request.Details.OpportunityType != kind {
		return usecase.SaveJobPosting{}, errors.New("opportunity_type must be internship for this route")
	}
	details := request.Details.details(kind)
	details.Internship = request.Details.Internship
	return usecase.SaveJobPosting{Title: request.Title, Description: request.Description, Details: details}, nil
}

func decodeJobPosting(decoder *json.Decoder, request any) error {
	if err := decoder.Decode(request); err != nil {
		return fmt.Errorf("invalid job posting JSON: %w", err)
	}
	if err := decoder.Decode(new(json.RawMessage)); !errors.Is(err, io.EOF) {
		return errors.New("expected a single JSON job posting")
	}
	return nil
}
