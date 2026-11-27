package domainerrors

import "errors"

var (
	ErrEmployerNotApproved       = errors.New("employer verification is not approved")
	ErrInvalidJobPosting         = errors.New("invalid job posting")
	ErrJobPostingNotFound        = errors.New("job posting not found")
	ErrInvalidOpportunityFilters = errors.New("invalid opportunity filters")
)
