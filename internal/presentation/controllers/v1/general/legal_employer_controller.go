package general

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
)

type LegalEmployerController struct {
	service usecase.LegalEmployerService
}

func NewLegalEmployerController(service usecase.LegalEmployerService) *LegalEmployerController {
	return &LegalEmployerController{service: service}
}

// Options godoc
// @Summary Get legal employer signup choices
// @Tags Legal employer
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /legal-employers/signup-options [get]
func (c *LegalEmployerController) Options(ctx *gin.Context) {
	locations := make(map[enums.Province][]enums.City)
	for _, city := range enums.Cities() {
		locations[city.Province()] = append(locations[city.Province()], city)
	}
	ctx.JSON(http.StatusOK, gin.H{"provinces": enums.Provinces(), "cities_by_province": locations, "genders": enums.Genders(), "legal_entity_types": enums.LegalEntityTypes(), "representative_positions": enums.RepresentativePositions(), "activities": enums.Interests()})
}

// RequestOTP godoc
// @Summary Resend legal employer verification OTP
// @Description Requires a complete pending signup submitted within the last 30 minutes. Signup sends the first code automatically. Resends a real SMS after a 60-second cooldown; the code expires in 5 minutes.
// @Tags Legal employer
// @Accept json
// @Produce json
// @Param request body ResendOTPRequest true "Phone number"
// @Success 202 {object} OTPResponse
// @Failure 400,409,429,500,503 {object} ErrorResponse
// @Router /legal-employers/otp [post]
func (c *LegalEmployerController) RequestOTP(ctx *gin.Context) {
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
// @Description Validates and saves the complete multipart submission and private files, then sends OTP automatically. Returns 202; no employer account exists until verify-otp succeeds. Pending signup lasts 30 minutes. Email and website are optional. Select activities from signup-options. Activity license is required when requires_activity_license is true. An attorney must upload authorization_document. Company identifier format is checked; official company and representative authority verification remains pending review. Password requires 12-72 bytes with uppercase, lowercase, number, symbol. Documents: PDF/PNG/JPEG up to 5 MiB each. Video: MP4/MOV, 1-20 seconds, up to 50 MiB. Requires configured S3 and ffprobe.
// @Tags Legal employer
// @Accept multipart/form-data
// @Produce json
// @Param company_name formData string true "Registered company or organization name"
// @Param legal_entity_type formData string true "Legal entity type" Enums(limited_liability,private_joint_stock,public_joint_stock,cooperative,institution,organization)
// @Param company_national_id formData string true "Eleven-digit company national identifier"
// @Param registration_number formData string true "Registration number (1-30 digits)"
// @Param economic_code formData string true "Twelve-digit economic code"
// @Param province formData string true "Province from signup-options"
// @Param city formData string true "City belonging to selected province"
// @Param address formData string true "Registered office address"
// @Param postal_code formData string true "Ten-digit postal code"
// @Param representative_first_name formData string true "Representative first name"
// @Param representative_last_name formData string true "Representative last name"
// @Param representative_position formData string true "Representative authority; attorney requires authorization_document" Enums(ceo,authorized_signatory,attorney)
// @Param representative_gender formData string true "Representative gender" Enums(female,male)
// @Param representative_national_code formData string true "Representative ten-digit national code with valid checksum"
// @Param has_tax_file formData bool false "Organization has a tax file" default(false)
// @Param activities formData []string true "Select 1-10 distinct activity areas from signup-options; repeat this form key for each choice" collectionFormat(multi)
// @Param website formData string false "HTTP or HTTPS website/social page URL"
// @Param requires_activity_license formData bool false "Whether the organization's activity requires a license; requires activity_license when true" default(false)
// @Param phone formData string true "Representative phone to receive verification OTP"
// @Param email formData string false "Email address"
// @Param password formData string true "Password: 12-72 bytes, uppercase, lowercase, digit and symbol"
// @Param establishment_document formData file true "Establishment notice or official gazette (PDF/PNG/JPEG, <=5 MiB)"
// @Param identity_document formData file true "Representative first page of identity document (PDF/PNG/JPEG, <=5 MiB)"
// @Param activity_license formData file false "License; required when requires_activity_license=true (PDF/PNG/JPEG, <=5 MiB)"
// @Param authorization_document formData file false "Power of attorney; required for attorney position (PDF/PNG/JPEG, <=5 MiB)"
// @Param verification_video formData file true "Representative identity video (MP4/MOV, 1-20 seconds, <=50 MiB)"
// @Success 202 {object} OTPResponse
// @Failure 400,409,413,429,500,503 {object} ErrorResponse
// @Router /legal-employers/signup [post]
func (c *LegalEmployerController) Signup(ctx *gin.Context) {
	if !parseEmployerMultipart(ctx, 76*1024*1024) {
		return
	}
	defer ctx.Request.MultipartForm.RemoveAll()
	var request LegalEmployerSignupRequest
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
	establishment, err := files.open(ctx, "establishment_document", true)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	license, err := files.open(ctx, "activity_license", false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	authorization, err := files.open(ctx, "authorization_document", false)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}
	in.EstablishmentDocument, in.IdentityDocument, in.VerificationVideo, in.ActivityLicense, in.AuthorizationDocument = *establishment, *identity, *video, license, authorization
	requestCtx, cancel := context.WithTimeout(ctx.Request.Context(), 3*time.Minute)
	defer cancel()
	if err := c.service.Signup(requestCtx, in); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent; verify to complete employer signup", ExpiresIn: 300, ResendAfter: 60})
}

// VerifyOTP godoc
// @Summary Verify legal employer and complete signup
// @Description Checks phone and code against the stored submission. Creates the shared employer and legal profile atomically and consumes the OTP. Does not require re-uploading files. Login is available after successful verification.
// @Tags Legal employer
// @Accept json
// @Produce json
// @Param request body VerifyOTPRequest true "Phone and received OTP"
// @Success 201 {object} LegalEmployerResponse
// @Failure 400,401,409,500 {object} ErrorResponse
// @Router /legal-employers/verify-otp [post]
func (c *LegalEmployerController) VerifyOTP(ctx *gin.Context) {
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
	ctx.Header("Cache-Control", "no-store")
	ctx.JSON(http.StatusCreated, LegalEmployerResponse{Data: identityFromLegalEmployer(employer)})
}

// Login godoc
// @Summary Login legal employer using phone and password
// @Description Returns an legal_employer JWT; job seeker tokens cannot access employer endpoints. Five failed attempts in 15 minutes trigger lockout. Pending-review employers can login; document review is a separate administrative process.
// @Tags Legal employer
// @Accept json
// @Produce json
// @Param request body LoginRequest true "Phone and password"
// @Success 200 {object} LegalEmployerLoginResponse
// @Failure 400,401,429,500 {object} ErrorResponse
// @Router /legal-employers/login [post]
func (c *LegalEmployerController) Login(ctx *gin.Context) {
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
	ctx.JSON(http.StatusOK, LegalEmployerLoginResponse{AccessTokenResponse: AccessTokenResponse{AccessToken: result.AccessToken.Value, TokenType: "Bearer", ExpiresAt: result.AccessToken.ExpiresAt}, Data: identityFromLegalEmployer(result.Employer)})
}

// Me godoc
// @Summary Get authenticated legal employer identity
// @Tags Legal employer
// @Security BearerAuth
// @Produce json
// @Success 200 {object} LegalEmployerResponse
// @Failure 401,500 {object} ErrorResponse
// @Router /legal-employers/me [get]
func (c *LegalEmployerController) Me(ctx *gin.Context) {
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
	ctx.JSON(http.StatusOK, LegalEmployerResponse{Data: identityFromLegalEmployer(employer)})
}

// Logout godoc
// @Summary Logout current legal employer token
// @Tags Legal employer
// @Security BearerAuth
// @Success 204 "Logged out"
// @Failure 401,500 {object} ErrorResponse
// @Router /legal-employers/logout [post]
func (c *LegalEmployerController) Logout(ctx *gin.Context) {
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
// @Tags Legal employer
// @Security BearerAuth
// @Produce json
// @Param kind path string true "Document kind" Enums(establishment,identity,activity-license,authorization,video)
// @Success 200 {object} DocumentURLResponse
// @Failure 400,401,500,503 {object} ErrorResponse
// @Router /legal-employers/documents/{kind} [get]
func (c *LegalEmployerController) DocumentURL(ctx *gin.Context) {
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
