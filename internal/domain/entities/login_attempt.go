package entities

import "time"

type LoginAttempt struct {
	Phone           string
	Failures        int
	WindowStartedAt time.Time
}
