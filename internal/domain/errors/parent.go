package domainerrors

import "errors"

var (
	ErrParentRequestNotFound   = errors.New("parent connection request not found or expired")
	ErrInvalidParent           = errors.New("invalid parent information")
	ErrInvalidInvitation       = errors.New("invalid, expired, or already used parent invitation code")
	ErrParentRequired          = errors.New("parent approval is mandatory for job seekers under 18 years")
	ErrParentAlreadyRegistered = errors.New("parent already registered; use the registered-parent approval endpoint")
	ErrParentNotRegistered     = errors.New("parent is not registered; use the parent invitation endpoint")
	ErrParentAlreadyConnected  = errors.New("parent already connected")
)
