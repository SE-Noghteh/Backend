package entities

// PendingEmployerSignup holds validated information and private object references
// until the phone is verified. Passwords must already be hashed.
type PendingEmployerSignup[T any] struct {
	Challenge    *SignupOTP
	Registration *T
}

type PendingLegalEmployerSignup = PendingEmployerSignup[LegalEmployer]
