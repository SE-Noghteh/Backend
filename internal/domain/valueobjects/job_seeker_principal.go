package valueobjects

import "time"

type JobSeekerPrincipal struct {
	ID        uint64
	TokenID   string
	ExpiresAt time.Time
}
