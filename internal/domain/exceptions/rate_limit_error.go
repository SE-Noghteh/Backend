package exceptions

import "zedlancer/backend/bootstrap"

type RateLimitError struct {
	Type string
}

func (e RateLimitError) Error() string {
	return "rate limit error occurred"
}

func NewRequestRateLimitError() *RateLimitError {
	return &RateLimitError{
		Type: bootstrap.Run().Constants.ErrorTags.RateLimit,
	}
}
