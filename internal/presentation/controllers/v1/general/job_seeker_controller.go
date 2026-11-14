package general

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/presentation/middleware"
)

type JobSeekerController struct{ service usecase.JobSeekerService }

func NewJobSeekerController(service usecase.JobSeekerService) *JobSeekerController {
	return &JobSeekerController{service: service}
}

// Options godoc
// @Summary Get signup choices
// @Description Fetch the predefined Persian cities, interests, and genders before signup.
// @Tags Job seeker signup
// @Produce json
// @Success 200 {object} SignupOptionsResponse
// @Router /job-seekers/signup-options [get]
func (c *JobSeekerController) Options(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, SignupOptionsResponse{Cities: enums.Cities(), Interests: enums.Interests(), Genders: []enums.Gender{enums.GenderFemale, enums.GenderMale}})
}

// RequestOTP godoc
// @Summary Resend signup OTP
// @Description Requires a pending validated signup submitted within the past 30 minutes. Resends a real SMS. Code expires in 300 seconds; wait 60 seconds before requesting another. Each code allows five failed verification attempts.
// @Tags Job seeker signup
// @Accept json
// @Produce json
// @Param request body ResendOTPRequest true "Your mobile number"
// @Success 202 {object} OTPResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Header 429 {string} Retry-After "Seconds until retry"
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /job-seekers/otp [post]
func (c *JobSeekerController) RequestOTP(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in ResendOTPRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON body"})
		return
	}
	if err := c.service.RequestOTP(ctx.Request.Context(), in.Phone); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent", ExpiresIn: 300, ResendAfter: 60})
}

// Signup godoc
// @Summary Start signup and automatically send OTP
// @Description Submit the complete profile, password and required private verification video before OTP is sent. Video must be MP4/MOV, 1-20 seconds, up to 50 MiB. Pending signup lasts 30 minutes; then verify the received OTP. Birth date is Jalali. Password must be 8-72 bytes.
// @Tags Job seeker signup
// @Accept multipart/form-data
// @Produce json
// @Param first_name formData string true "First name"
// @Param last_name formData string true "Last name"
// @Param national_code formData string true "Ten-digit national code"
// @Param city formData string true "City from signup-options"
// @Param address formData string true "Address"
// @Param birth_year formData int true "Jalali birth year"
// @Param birth_month formData int true "Jalali birth month"
// @Param birth_day formData int true "Jalali birth day"
// @Param gender formData string true "Gender" Enums(female,male)
// @Param interests formData []string true "One to ten unique interests from signup-options" collectionFormat(multi)
// @Param phone formData string true "Phone number"
// @Param password formData string true "Password (8-72 bytes)"
// @Param verification_video formData file true "Verification video (MP4/MOV, 1-20 seconds, <=50 MiB)"
// @Success 202 {object} OTPResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 413 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /job-seekers/signup [post]
func (c *JobSeekerController) Signup(ctx *gin.Context) {
	if !parseEmployerMultipart(ctx, 51*1024*1024) {
		return
	}
	defer ctx.Request.MultipartForm.RemoveAll()
	var request JobSeekerSignupRequest
	if err := ctx.ShouldBind(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid form fields"})
		return
	}
	files := &employerMultipartFiles{}
	defer files.close()
	video, err := files.open(ctx, "verification_video", true)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	in := request.input()
	in.VerificationVideo = *video
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 3*time.Minute)
	defer cancel()
	if err := c.service.Signup(requestCtx, in); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent; verify to complete signup", ExpiresIn: 300, ResendAfter: 60})
}

// VideoURL godoc
// @Summary Get a private download link for your own verification video
// @Tags Job seeker authentication
// @Security BearerAuth
// @Produce json
// @Success 200 {object} DocumentURLResponse
// @Failure 400,401,500,503 {object} ErrorResponse
// @Router /job-seekers/documents/video [get]
func (c *JobSeekerController) VideoURL(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	link, err := c.service.VideoURL(ctx.Request.Context(), seeker.ID)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, DocumentURLResponse{URL: link, ExpiresIn: 300})
}

// VerifyOTP godoc
// @Summary Verify OTP and complete signup
// @Description Creates the account from the stored validated profile and consumes the OTP atomically. Use the same phone number as signup. After success, login with your phone and password to obtain a JWT.
// @Tags Job seeker signup
// @Accept json
// @Produce json
// @Param request body VerifyOTPRequest true "Phone and received code"
// @Success 201 {object} SignupResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /job-seekers/verify-otp [post]
func (c *JobSeekerController) VerifyOTP(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in VerifyOTPRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	seeker, err := c.service.VerifyOTP(ctx.Request.Context(), in.input())
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, SignupResponse{Data: identityFromJobSeeker(seeker)})
}
