package service

import (
	"context"
	"errors"
	"testing"
	"time"

	domainerrors "zedlancer/backend/internal/domain/errors"
	"zedlancer/backend/internal/domain/valueobjects"
)

func TestJobSeekerSignupAgeBoundaries(t *testing.T) {
	date := valueobjects.JalaliDate{Year: 1385, Month: 8, Day: 23}
	birth, err := date.Gregorian()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name    string
		now     time.Time
		valid   bool
		message string
	}{
		{"day before 15", birth.AddDate(15, 0, -1), false, "invalid signup: job seeker is under age: minimum age is 15 years"},
		{"15th birthday", birth.AddDate(15, 0, 0), true, ""},
		{"day after 15", birth.AddDate(15, 0, 1), true, ""},
		{"22nd birthday", birth.AddDate(22, 0, 0), true, ""},
		{"last day aged 22", birth.AddDate(23, 0, -1), true, ""},
		{"23rd birthday", birth.AddDate(23, 0, 0), false, "invalid signup: job seeker is over age: maximum age is 22 years"},
		{"future birth", birth.AddDate(-1, 0, 0), false, "invalid signup: job seeker is under age: minimum age is 15 years"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := validSignup()
			in.BirthDate = date
			err := validateSignupAt(&in, tc.now)
			if tc.valid {
				if err != nil {
					t.Fatal(err)
				}
			} else if !errors.Is(err, domainerrors.ErrInvalidSignup) || err.Error() != tc.message {
				t.Fatalf("expected clear age error, got %v", err)
			}
		})
	}
}

func TestJobSeekerInvalidAgeDoesNotUploadOrSendOTP(t *testing.T) {
	for _, year := range []int{1300, 1500} {
		m := &signupMemory{}
		storage := &employerMemory{}
		s := NewJobSeekerService(m, m, testPasswords{}, storage, storage)
		in := validSignup()
		in.BirthDate = valueobjects.JalaliDate{Year: year, Month: 1, Day: 1}
		if err := s.Signup(context.Background(), in); !errors.Is(err, domainerrors.ErrInvalidSignup) {
			t.Fatalf("invalid age accepted: %v", err)
		}
		if m.smsCount != 0 || storage.putCount != 0 || m.otp != nil || m.seeker != nil {
			t.Fatal("invalid age caused signup side effects")
		}
	}
}
