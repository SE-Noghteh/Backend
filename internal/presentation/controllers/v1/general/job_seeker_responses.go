package general

import (
	"time"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
)

type SignupOptionsResponse struct {
	Cities    []enums.City     `json:"cities"`
	Interests []enums.Interest `json:"interests"`
	Genders   []enums.Gender   `json:"genders"`
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
type ErrorResponse struct {
	Message string `json:"message" example:"invalid or expired OTP"`
}

func identityFromJobSeeker(seeker *entities.JobSeeker) SignupIdentity {
	return SignupIdentity{ID: seeker.ID, FirstName: seeker.FirstName, LastName: seeker.LastName, Phone: seeker.Phone, CreatedAt: seeker.CreatedAt}
}

type AccessTokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type" example:"Bearer"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type LoginResponse struct {
	AccessTokenResponse
	Data SignupIdentity `json:"data"`
}
