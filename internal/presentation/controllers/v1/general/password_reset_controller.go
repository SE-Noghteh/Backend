package general

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
)

type PasswordResetController struct {
	service usecase.PasswordResetService
}

func NewPasswordResetController(service usecase.PasswordResetService) *PasswordResetController {
	return &PasswordResetController{service: service}
}

type ForgotPasswordRequest struct {
	Phone string `json:"phone" binding:"required" example:"09123456789"`
}

type VerifyPasswordResetOTPRequest struct {
	Phone string `json:"phone" binding:"required" example:"09123456789"`
	OTP   string `json:"otp" binding:"required" example:"123456"`
}

type PasswordResetTokenResponse struct {
	ResetToken string `json:"reset_token"`
	ExpiresIn  int    `json:"expires_in" example:"300"`
}

type SetNewPasswordRequest struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

type PasswordResetResponse struct {
	Message string `json:"message" example:"Password reset successfully. Please log in with your new password."`
}

// ForgotPassword godoc
// @Summary Request a password reset code for the selected account type
// @Description Returns the same 202 response for unknown numbers and requests during the 60-second cooldown. The six-digit OTP expires after five minutes.
// @Tags Password reset
// @Accept json
// @Produce json
// @Param request body ForgotPasswordRequest true "Registered mobile number"
// @Success 202 {object} OTPResponse
// @Failure 400,500,503 {object} ErrorResponse
// @Router /job-seekers/forgot-password [post]
// @Router /parents/forgot-password [post]
// @Router /individual-employers/forgot-password [post]
// @Router /legal-employers/forgot-password [post]
func (c *PasswordResetController) ForgotPassword(role enums.AccountRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var in ForgotPasswordRequest
		if !bindParentJSON(ctx, &in) {
			return
		}
		if err := c.service.Request(ctx.Request.Context(), role, in.Phone); err != nil {
			respondAPIError(ctx, err)
			return
		}
		ctx.JSON(http.StatusAccepted, OTPResponse{Message: "If an account exists for this phone number, an OTP has been sent", ExpiresIn: 300, ResendAfter: 60})
	}
}

// VerifyOTP godoc
// @Summary Verify a password reset code before entering the new password
// @Description Consumes the OTP and returns a one-time reset token valid for five minutes. The token applies only to this account type.
// @Tags Password reset
// @Accept json
// @Produce json
// @Param request body VerifyPasswordResetOTPRequest true "Phone and password reset OTP"
// @Success 200 {object} PasswordResetTokenResponse
// @Failure 400,401,500 {object} ErrorResponse
// @Router /job-seekers/verify-reset-otp [post]
// @Router /parents/verify-reset-otp [post]
// @Router /individual-employers/verify-reset-otp [post]
// @Router /legal-employers/verify-reset-otp [post]
func (c *PasswordResetController) VerifyOTP(role enums.AccountRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var in VerifyPasswordResetOTPRequest
		if !bindParentJSON(ctx, &in) {
			return
		}
		token, err := c.service.VerifyOTP(ctx.Request.Context(), role, in.Phone, in.OTP)
		if err != nil {
			respondAPIError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, PasswordResetTokenResponse{ResetToken: token, ExpiresIn: 300})
	}
}

// ResetPassword godoc
// @Summary Set a new password using the verified reset token
// @Description Consumes the token and invalidates existing sessions. Use 8-72 bytes for job seekers and parents. Employers require 12-72 bytes with uppercase, lowercase, a digit and a symbol.
// @Tags Password reset
// @Accept json
// @Produce json
// @Param request body SetNewPasswordRequest true "Reset token and new password"
// @Success 200 {object} PasswordResetResponse
// @Failure 400,401,500 {object} ErrorResponse
// @Router /job-seekers/reset-password [post]
// @Router /parents/reset-password [post]
// @Router /individual-employers/reset-password [post]
// @Router /legal-employers/reset-password [post]
func (c *PasswordResetController) ResetPassword(role enums.AccountRole) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var in SetNewPasswordRequest
		if !bindParentJSON(ctx, &in) {
			return
		}
		if err := c.service.Reset(ctx.Request.Context(), role, in.ResetToken, in.NewPassword); err != nil {
			respondAPIError(ctx, err)
			return
		}
		ctx.JSON(http.StatusOK, PasswordResetResponse{Message: "Password reset successfully. Please log in with your new password."})
	}
}
