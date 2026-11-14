package general

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/presentation/middleware"
)

type PhoneChangeRequest struct {
	NewPhone string `json:"new_phone" binding:"required" example:"09129876543"`
}

type VerifyPhoneChangeRequest struct {
	NewPhone string `json:"new_phone" binding:"required" example:"09129876543"`
	OTP      string `json:"otp" binding:"required" example:"123456"`
}

type DeleteJobSeekerRequest struct {
	Password string `json:"password" binding:"required" example:"your-current-password"`
}

type DeleteJobSeekerResponse struct {
	Message string `json:"message" example:"Account deleted successfully"`
}

// RequestPhoneChange godoc
// @Summary Send an OTP to a new job seeker phone number
// @Description Requires login. Keeps the current number until verification succeeds. Repeat this request to resend after 60 seconds. Only the latest destination/code is valid. Code expires after five minutes and allows five failed attempts.
// @Tags Job seeker account
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body PhoneChangeRequest true "New mobile number"
// @Success 202 {object} OTPResponse
// @Failure 400,401,409,429,500,503 {object} ErrorResponse
// @Router /job-seekers/phone-change [post]
func (c *JobSeekerController) RequestPhoneChange(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in PhoneChangeRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	if err := c.service.RequestPhoneChange(ctx.Request.Context(), seeker.ID, in.NewPhone); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "OTP sent to new phone", ExpiresIn: 300, ResendAfter: 60})
}

// VerifyPhoneChange godoc
// @Summary Verify the new phone and change your number
// @Description Requires the same logged-in account that requested the code. Consumes the OTP atomically. Login subsequently uses the new number; existing access tokens remain valid.
// @Tags Job seeker account
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body VerifyPhoneChangeRequest true "New mobile number and received code"
// @Success 200 {object} SignupResponse
// @Failure 400,401,409,500 {object} ErrorResponse
// @Router /job-seekers/phone-change/verify [post]
func (c *JobSeekerController) VerifyPhoneChange(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in VerifyPhoneChangeRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	result, err := c.service.VerifyPhoneChange(ctx.Request.Context(), seeker.ID, usecase.VerifySignupOTP{Phone: in.NewPhone, OTP: in.OTP})
	if err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, SignupResponse{Data: identityFromJobSeeker(result)})
}

// DeleteAccount godoc
// @Summary Delete your job seeker account with password confirmation
// @Description Requires login and your current password. Permanently deletes the account and pending phone change, then removes its verification video. All existing tokens become unusable. Five failed password confirmations share the login throttle (15-minute window).
// @Tags Job seeker account
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param request body DeleteJobSeekerRequest true "Current password"
// @Success 200 {object} DeleteJobSeekerResponse
// @Failure 400,401,429,500 {object} ErrorResponse
// @Router /job-seekers/me [delete]
func (c *JobSeekerController) DeleteAccount(ctx *gin.Context) {
	seeker, ok := middleware.CurrentJobSeeker(ctx)
	if !ok {
		ctx.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Message: "authentication required"})
		return
	}
	ctx.Header("Cache-Control", "no-store")
	ctx.Request.Body = http.MaxBytesReader(ctx.Writer, ctx.Request.Body, 16*1024)
	var in DeleteJobSeekerRequest
	if err := ctx.ShouldBindJSON(&in); err != nil {
		ctx.JSON(http.StatusBadRequest, ErrorResponse{Message: "invalid JSON body"})
		return
	}
	if err := c.service.DeleteAccount(ctx.Request.Context(), seeker.ID, in.Password); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, DeleteJobSeekerResponse{Message: "Account deleted successfully"})
}
