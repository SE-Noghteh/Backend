package general

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

type ForgotJobSeekerPasswordRequest struct {
	Phone string `json:"phone" binding:"required" example:"09123456789"`
}
type ResetJobSeekerPasswordRequest struct {
	Phone       string `json:"phone" binding:"required" example:"09123456789"`
	OTP         string `json:"otp" binding:"required" example:"123456"`
	NewPassword string `json:"new_password" binding:"required" example:"your-new-password"`
}
type PasswordResetResponse struct {
	Message string `json:"message" example:"Password reset successfully. Please log in with your new password."`
}

// ForgotPassword godoc
// @Summary Send a job seeker password reset OTP
// @Description Public endpoint. Sends an OTP to the registered, verified phone. Repeat to resend after 60 seconds. Codes expire after five minutes and permit five failed attempts. Unknown or unverified phones receive the same 202 response without sending SMS. Reset OTPs are separate from signup and phone-change OTPs.
// @Tags Job seekers
// @Accept json
// @Produce json
// @Param request body ForgotJobSeekerPasswordRequest true "Registered job seeker phone"
// @Success 202 {object} OTPResponse
// @Failure 400,429,500,503 {object} ErrorResponse
// @Router /job-seekers/forgot-password [post]
func (c *JobSeekerController) ForgotPassword(ctx *gin.Context) {
	var in ForgotJobSeekerPasswordRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	if err := c.service.ForgotPassword(ctx.Request.Context(), in.Phone); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusAccepted, OTPResponse{Message: "If an account exists for this phone number, an OTP has been sent", ExpiresIn: 300, ResendAfter: 60})
}

// ResetPassword godoc
// @Summary Reset a job seeker password using phone and OTP
// @Description Public endpoint. Submit the registered phone, its password-reset OTP and a new password (8-72 bytes). Atomically consumes the OTP, replaces the bcrypt password, clears login failures, invalidates all existing job seeker login tokens and cancels pending phone-change challenges. Login again with the new password. Approval and parent connection are preserved.
// @Tags Job seekers
// @Accept json
// @Produce json
// @Param request body ResetJobSeekerPasswordRequest true "Phone, reset OTP and new password"
// @Success 200 {object} PasswordResetResponse
// @Failure 400,401,500 {object} ErrorResponse
// @Router /job-seekers/reset-password [post]
func (c *JobSeekerController) ResetPassword(ctx *gin.Context) {
	var in ResetJobSeekerPasswordRequest
	if !bindParentJSON(ctx, &in) {
		return
	}
	if err := c.service.ResetPassword(ctx.Request.Context(), in.Phone, in.OTP, in.NewPassword); err != nil {
		respondAPIError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, PasswordResetResponse{Message: "Password reset successfully. Please log in with your new password."})
}
