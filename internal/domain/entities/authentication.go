package entities

import "time"

type AccessToken struct {
	Value     string
	ExpiresAt time.Time
}

type JobSeekerPrincipal struct {
	ID        uint64
	TokenID   string
	ExpiresAt time.Time
}

type LoginAttempt struct {
	Phone           string
	Failures        int
	WindowStartedAt time.Time
}
