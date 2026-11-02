package domainerrors

import "errors"

var (
	ErrInvalidSignup      = errors.New("invalid signup")
	ErrInvalidOTP         = errors.New("invalid or expired OTP")
	ErrSignupConflict     = errors.New("phone or national code already registered")
	ErrOTPTooSoon         = errors.New("wait before requesting another OTP")
	ErrSMSUnavailable     = errors.New("SMS service unavailable")
	ErrInvalidCredentials = errors.New("invalid phone number or password")
	ErrUnauthenticated    = errors.New("invalid or expired access token")
	ErrLoginThrottled     = errors.New("too many login attempts; try again in 15 minutes")
	ErrNoPendingSignup    = errors.New("submit valid signup information before requesting an OTP")
)
