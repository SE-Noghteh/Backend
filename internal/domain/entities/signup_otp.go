package entities

import "time"

type SignupOTP struct {
	Phone        string
	Salt         string
	CodeHash     string
	Attempts     int
	SentAt       time.Time
	ExpiresAt    time.Time
	Registration *JobSeeker
}
