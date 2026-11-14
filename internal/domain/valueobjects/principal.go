package valueobjects

import "time"

type Principal struct {
	ID        uint64
	TokenID   string
	ExpiresAt time.Time
}
