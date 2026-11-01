package general

import (
	"time"
	"zedlancer/backend/internal/domain/entities"
)

type RequestOTPBody struct {
	Phone string `json:"phone" example:"09123456789" validate:"required"`
}

type SignupOptionsResponse struct {
	Cities    []entities.City     `json:"cities"`
	Interests []entities.Interest `json:"interests"`
	Genders   []entities.Gender   `json:"genders"`
}

type OTPResponse struct {
	Message     string `json:"message" example:"OTP sent"`
	ExpiresIn   int    `json:"expires_in" example:"300"`
	ResendAfter int    `json:"resend_after" example:"60"`
}

type SignupIdentity struct {
	ID        uint64    `json:"id" example:"1"`
	FirstName string    `json:"first_name" example:"علی"`
	LastName  string    `json:"last_name" example:"رضایی"`
	Phone     string    `json:"phone" example:"09123456789"`
	CreatedAt time.Time `json:"created_at"`
}

type SignupResponse struct {
	Data SignupIdentity `json:"data"`
}
type SignupErrorResponse struct {
	Message string `json:"message" example:"invalid or expired OTP"`
}

func identityFromJobSeeker(seeker *entities.JobSeeker) SignupIdentity {
	return SignupIdentity{ID: seeker.ID, FirstName: seeker.FirstName, LastName: seeker.LastName, Phone: seeker.Phone, CreatedAt: seeker.CreatedAt}
}

type LoginResponse struct {
	AccessToken string         `json:"access_token"`
	TokenType   string         `json:"token_type" example:"Bearer"`
	ExpiresAt   time.Time      `json:"expires_at"`
	Data        SignupIdentity `json:"data"`
}
