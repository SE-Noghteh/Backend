package valueobjects

import "time"

type Principal struct {
	SessionVersion uint64
	ID             uint64
	TokenID        string
	ExpiresAt      time.Time
}
