package domainerrors

import "errors"

var (
	ErrInvalidParent     = errors.New("invalid parent information")
	ErrInvalidInvitation = errors.New("invalid, expired, or already used parent invitation code")
	ErrParentRequired    = errors.New("parent approval is mandatory for job seekers under 18 years")
)
