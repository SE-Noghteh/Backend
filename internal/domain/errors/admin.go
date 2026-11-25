package domainerrors

import "errors"

var (
	ErrInvalidAdminCredentials = errors.New("invalid username or password")
	ErrInvalidReview           = errors.New("invalid admin review")
	ErrReviewNotFound          = errors.New("account not found")
	ErrReviewDocumentNotFound  = errors.New("review document not found")
	ErrSignupIncomplete        = errors.New("job seekers under 18 must connect a parent before admin review")
)
