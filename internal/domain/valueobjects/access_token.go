package valueobjects

import "time"

type AccessToken struct {
	Value     string
	ExpiresAt time.Time
}
