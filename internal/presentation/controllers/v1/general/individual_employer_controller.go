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

type IndividualEmployerController struct {
	service usecase.IndividualEmployerService
}

func NewIndividualEmployerController(service usecase.IndividualEmployerService) *IndividualEmployerController {
	return &IndividualEmployerController{service: service}
}

// Options godoc
// @Summary Get individual employer signup choices
// @Tags Individual employer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /individual-employers/signup-options [get]
func (c *IndividualEmployerController) Options(ctx *gin.Context) {
	locations := make(map[enums.Province][]enums.City)
	for _, city := range enums.Cities() {
		locations[city.Province()] = append(locations[city.Province()], city)
	}
	ctx.JSON(http.StatusOK, gin.H{"provinces": enums.Provinces(), "cities_by_province": locations, "genders": enums.Genders()})
}

// RequestOTP godoc
// @Summary Resend individual employer verification OTP
// @Description Requires a complete pending signup submitted within the last 30 minutes. Signup sends the first code automatically. Resends a real SMS after a 60-second cooldown; the code expires in 5 minutes.
// @Tags Individual employer
// @Accept json
// @Produce json
// @Param request body ResendOTPRequest true "Phone number"
// @Success 202 {object} OTPResponse
// @Failure 400,409,429,500,503 {object} ErrorResponse
// @Router /individual-employers/otp [post]
func (c *IndividualEmployerController) RequestOTP(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in ResendOTPRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	if err := c.service.RequestOTP(ctx.Request.Context(), in.Phone); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent", ExpiresIn: 300, ResendAfter: 60})
}

// Signup godoc
// @Summary Submit employer information and automatically send verification OTP
// @Description Validates and saves the complete multipart submission and private files, then sends OTP automatically. Returns 202; no employer account exists until verify-otp succeeds. Pending signup lasts 30 minutes. Business fields, email and business license are optional. Password requires 12-72 bytes with uppercase, lowercase, number, symbol. Documents: PDF/PNG/JPEG up to 5 MiB each. Video: MP4/MOV, 30-60 seconds, up to 50 MiB. Requires configured S3 and ffprobe.
// @Tags Individual employer
// @Accept multipart/form-data
// @Produce json
// @Param first_name formData string true "First name"
// @Param last_name formData string true "Last name"
// @Param father_name formData string true "Father name"
// @Param national_code formData string true "Ten-digit national code with valid checksum"
// @Param gender formData string true "Gender" Enums(female,male)
// @Param birth_year formData int true "Jalali year" default(1370)
// @Param birth_month formData int true "Jalali month" default(1)
// @Param birth_day formData int true "Jalali day" default(1)
// @Param province formData string true "Province from signup-options"
// @Param city formData string true "City belonging to selected province"
// @Param address formData string true "Full residence address"
// @Param postal_code formData string true "Ten-digit postal code"
// @Param business_name formData string false "Business or trade name"
// @Param business_activity formData string false "Business activity"
// @Param website formData string false "HTTP or HTTPS website/social page URL"
// @Param has_legal_guardian formData bool false "Has legal guardian" default(false)
// @Param phone formData string true "Phone to receive verification OTP"
// @Param email formData string false "Email address"
// @Param password formData string true "Password (12-72 bytes)"
// @Param identity_document formData file true "First page of identity document (PDF/PNG/JPEG, <=5 MiB)"
// @Param business_license formData file false "Optional business license (PDF/PNG/JPEG, <=5 MiB)"
// @Param verification_video formData file true "Identity video (MP4/MOV, 30-60 seconds, <=50 MiB)"
// @Success 202 {object} OTPResponse
// @Failure 400,409,413,429,500,503 {object} ErrorResponse
// @Router /individual-employers/signup [post]
func (c *IndividualEmployerController) Signup(ctx *gin.Context) {
	if !parseEmployerMultipart(ctx, 61*1024*1024) {
		return
	}
	defer ctx.Request.MultipartForm.RemoveAll()
	var request IndividualEmployerSignupRequest
	if err := ctx.ShouldBind(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid form fields"})
		return
	}
	in := request.input()
	files := &employerMultipartFiles{}
	defer files.close()
	identity, err := files.open(ctx, "identity_document", true)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	video, err := files.open(ctx, "verification_video", true)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	license, err := files.open(ctx, "business_license", false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	in.IdentityDocument, in.VerificationVideo, in.BusinessLicense = *identity, *video, license
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 3*time.Minute)
	defer cancel()
	if err := c.service.Signup(requestCtx, in); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent; verify to complete employer signup", ExpiresIn: 300, ResendAfter: 60})
}

// VerifyOTP godoc
// @Summary Verify individual employer and complete signup
// @Description Checks phone and code against the stored submission. Creates the shared employer and individual profile atomically and consumes the OTP. Does not require re-uploading files. Login is available after successful verification.
// @Tags Individual employer
// @Accept json
// @Produce json
// @Param request body VerifyOTPRequest true "Phone and received OTP"
// @Success 201 {object} IndividualEmployerResponse
// @Failure 400,401,409,500 {object} ErrorResponse
// @Router /individual-employers/verify-otp [post]
func (c *IndividualEmployerController) VerifyOTP(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in VerifyOTPRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	employer, err := c.service.VerifyOTP(ctx.Request.Context(), in.input())
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, IndividualEmployerResponse{Data: identityFromIndividualEmployer(employer)})
}

// Login godoc
// @Summary Login individual employer using phone and password
// @Description Returns an individual_employer JWT; job seeker tokens cannot access employer endpoints. Five failed attempts in 15 minutes trigger lockout. Pending-review employers can login; document review is a separate administrative process.
// @Tags Individual employer
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Phone and password"
// @Success 200 {object} IndividualEmployerLoginResponse
// @Failure 400,401,429,500 {object} ErrorResponse
// @Router /individual-employers/login [post]
func (c *IndividualEmployerController) Login(ctx *gin.Context) {
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in LoginRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	result, err := c.service.Login(ctx.Request.Context(), in.Phone, in.Password)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, IndividualEmployerLoginResponse{AccessTokenResponse: AccessTokenResponse{AccessToken: result.AccessToken.Value, TokenType: "Bearer", ExpiresAt: result.AccessToken.ExpiresAt}, Data: identityFromIndividualEmployer(result.Employer)})
}

func employerBearer(ctx *gin.Context) (string, bool) {
	token, ok := middleware.BearerToken(ctx)
	if !ok {
		ctx.Header("WWW-Authenticate", "Bearer")
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Message: "Authorization header required: Bearer <access_token>"})
	}
	return token, ok
}

// Me godoc
// @Summary Get authenticated individual employer identity
// @Tags Individual employer
// @Security BearerAuth
// @Produce json
// @Success 200 {object} IndividualEmployerResponse
// @Failure 401,500 {object} ErrorResponse
// @Router /individual-employers/me [get]
func (c *IndividualEmployerController) Me(ctx *gin.Context) {
	token, ok := employerBearer(ctx)
	if !ok {
		return
	}
	employer, err := c.service.Authenticate(ctx.Request.Context(), token)
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, IndividualEmployerResponse{Data: identityFromIndividualEmployer(employer)})
}

// Logout godoc
// @Summary Logout current individual employer token
// @Tags Individual employer
// @Security BearerAuth
// @Success 204 "Logged out"
// @Failure 401,500 {object} ErrorResponse
// @Router /individual-employers/logout [post]
func (c *IndividualEmployerController) Logout(ctx *gin.Context) {
	token, ok := employerBearer(ctx)
	if !ok {
		return
	}
	if err := c.service.Logout(ctx.Request.Context(), token); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Status(http.StatusNoContent)
}

// DocumentURL godoc
// @Summary Get private download link for your own employer document
// @Description The signed link expires in five minutes. No public S3 URLs or credentials are exposed.
// @Tags Individual employer
// @Security BearerAuth
// @Produce json
// @Param kind path string true "Document kind" Enums(identity,business-license,video)
// @Success 200 {object} DocumentURLResponse
// @Failure 400,401,500,503 {object} ErrorResponse
// @Router /individual-employers/documents/{kind} [get]
func (c *IndividualEmployerController) DocumentURL(ctx *gin.Context) {
	token, ok := employerBearer(ctx)
	if !ok {
		return
	}
	url, err := c.service.DocumentURL(ctx.Request.Context(), token, ctx.Param("kind"))
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusOK, DocumentURLResponse{URL: url, ExpiresIn: 300})
}
