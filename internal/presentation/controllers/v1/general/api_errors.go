package general

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	domainerrors "zedlancer/backend/internal/domain/errors"
)

func respondAPIError(ctx *gin.Context, err error) {
	status, message := http.StatusInternalServerError, "internal server error"
	switch {
	case errors.Is(err, domainerrors.ErrInvalidSignup):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, domainerrors.ErrInvalidUpload):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, domainerrors.ErrStorageUnavailable):
		status, message = http.StatusServiceUnavailable, err.Error()
	case errors.Is(err, domainerrors.ErrNoPendingSignup):
		status, message = http.StatusBadRequest, err.Error()
	case errors.Is(err, domainerrors.ErrInvalidCredentials), errors.Is(err, domainerrors.ErrUnauthenticated):
		status, message = http.StatusUnauthorized, err.Error()
		ctx.Header("WWW-Authenticate", "Bearer")
	case errors.Is(err, domainerrors.ErrLoginThrottled):
		status, message = http.StatusTooManyRequests, err.Error()
		ctx.Header("Retry-After", "900")
	case errors.Is(err, domainerrors.ErrInvalidOTP):
		status, message = http.StatusUnauthorized, err.Error()
	case errors.Is(err, domainerrors.ErrSignupConflict):
		status, message = http.StatusConflict, err.Error()
	case errors.Is(err, domainerrors.ErrOTPTooSoon):
		status, message = http.StatusTooManyRequests, err.Error()
		ctx.Header("Retry-After", "60")
	case errors.Is(err, domainerrors.ErrSMSUnavailable):
		status, message = http.StatusServiceUnavailable, err.Error()
	}
	ctx.JSON(status, gin.H{"message": message})
}
