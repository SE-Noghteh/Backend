package general

import (
	"errors"
	"github.com/gin-gonic/gin"
	"net/http"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

type JobSeekerController struct{ service usecase.JobSeekerService }

func NewJobSeekerController(service usecase.JobSeekerService) *JobSeekerController {
	return &JobSeekerController{service: service}
}

func signupError(ctx *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, ports.ErrInvalidSignup):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, ports.ErrNoPendingSignup):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, ports.ErrInvalidCredentials), errors.Is(err, ports.ErrUnauthenticated):
		status, message = http.StatusUnauthorized, err.Error()
		ctx.Header("WWW-Authenticate", "Bearer")
	case errors.Is(err, ports.ErrLoginThrottled):
		status, message = http.StatusTooManyRequests, err.Error()
		ctx.Header("Retry-After", "900")
	case errors.Is(err, ports.ErrInvalidOTP):
		status, message = http.StatusUnauthorized, err.Error()
	case errors.Is(err, ports.ErrSignupConflict):
		status, message = http.StatusConflict, err.Error()
	case errors.Is(err, ports.ErrOTPTooSoon):
		status, message = http.StatusTooManyRequests, err.Error()
		ctx.Header("Retry-After", "60")
	case errors.Is(err, ports.ErrSMSUnavailable):
		status, message = http.StatusServiceUnavailable, err.Error()
	}
	ctx.JSON(status, gin.H{"message": message})
}

// Options godoc
// @Summary Get signup choices
// @Description Fetch the predefined Persian cities, interests, and genders before signup.
// @Tags Job seeker signup
// @Produce json
// @Success 200 {object} SignupOptionsResponse
// @Router /job-seekers/signup-options [get]
func (c *JobSeekerController) Options(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, SignupOptionsResponse{Cities: entities.Cities(), Interests: entities.Interests(), Genders: []entities.Gender{entities.GenderFemale, entities.GenderMale}})
}

// RequestOTP godoc
// @Summary Resend signup OTP
// @Description Requires a pending validated signup submitted within the past 30 minutes. Resends a real SMS. Code expires in 300 seconds; wait 60 seconds before requesting another. Each code allows five failed verification attempts.
// @Tags Job seeker signup
// @Accept json
// @Produce json
// @Param request body RequestOTPBody true "Your mobile number"
// @Success 202 {object} OTPResponse
// @Failure 400 {object} SignupErrorResponse
// @Failure 409 {object} SignupErrorResponse
// @Failure 429 {object} SignupErrorResponse
// @Header 429 {string} Retry-After "Seconds until retry"
// @Failure 500 {object} SignupErrorResponse
// @Failure 503 {object} SignupErrorResponse
// @Router /job-seekers/otp [post]
func (c *JobSeekerController) RequestOTP(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in RequestOTPBody
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON body"})
		return
	}
	if err := c.service.RequestOTP(ctx.Request.Context(), in.Phone); err != nil {
		signupError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent", ExpiresIn: 300, ResendAfter: 60})
}

// Signup godoc
// @Summary Start signup and automatically send OTP
// @Description Validates the whole profile and password before sending a real SMS. Stores a pending registration for 30 minutes; no account is created yet. Next call verify-otp with your phone and received code. Birth date is Jalali. Select 1-10 unique interests. Password must be 8-72 bytes and is stored only as a hash.
// @Tags Job seeker signup
// @Accept json
// @Produce json
// @Param request body usecase.SignupJobSeeker true "Profile and password"
// @Success 202 {object} OTPResponse
// @Failure 400 {object} SignupErrorResponse
// @Failure 409 {object} SignupErrorResponse
// @Failure 429 {object} SignupErrorResponse
// @Failure 500 {object} SignupErrorResponse
// @Failure 503 {object} SignupErrorResponse
// @Router /job-seekers/signup [post]
func (c *JobSeekerController) Signup(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in usecase.SignupJobSeeker
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"message": "invalid JSON body"})
		return
	}
	if err := c.service.Signup(ctx.Request.Context(), in); err != nil {
		signupError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent; verify to complete signup", ExpiresIn: 300, ResendAfter: 60})
}

// VerifyOTP godoc
// @Summary Verify OTP and complete signup
// @Description Creates the account from the stored validated profile and consumes the OTP atomically. Use the same phone number as signup. After success, login with your phone and password to obtain a JWT.
// @Tags Job seeker signup
// @Accept json
// @Produce json
// @Param request body usecase.VerifySignupOTP true "Phone and received code"
// @Success 201 {object} SignupResponse
// @Failure 400 {object} SignupErrorResponse
// @Failure 401 {object} SignupErrorResponse
// @Failure 409 {object} SignupErrorResponse
// @Failure 500 {object} SignupErrorResponse
// @Router /job-seekers/verify-otp [post]
func (c *JobSeekerController) VerifyOTP(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in usecase.VerifySignupOTP
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, SignupErrorResponse{Message: "invalid JSON body"})
		return
	}
	seeker, err := c.service.VerifyOTP(ctx.Request.Context(), in)
	if err != nil {
		signupError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, SignupResponse{Data: identityFromJobSeeker(seeker)})
}
