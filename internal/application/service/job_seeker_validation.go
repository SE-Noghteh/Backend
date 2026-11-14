package service

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

func validateSignup(in *usecase.SignupJobSeeker) error {
	in.FirstName, in.LastName, in.Address = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName), strings.TrimSpace(in.Address)
	invalid := func(field string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidSignup, field) }
	if in.FirstName == "" || utf8.RuneCountInString(in.FirstName) > 100 {
		return invalid("first_name")
	}
	if in.LastName == "" || utf8.RuneCountInString(in.LastName) > 100 {
		return invalid("last_name")
	}
	if in.Address == "" || utf8.RuneCountInString(in.Address) > 1000 {
		return invalid("address")
	}
	if !validNationalCode(in.NationalCode) {
		return invalid("national_code")
	}
	if len(in.Password) < 8 || len(in.Password) > 72 {
		return invalid("password must contain 8 to 72 bytes")
	}
	if !in.City.Valid() {
		return invalid("city")
	}
	if !in.Gender.Valid() {
		return invalid("gender")
	}
	if len(in.Interests) < 1 || len(in.Interests) > 10 {
		return invalid("interests")
	}
	seen := map[enums.Interest]bool{}
	for _, interest := range in.Interests {
		if !interest.Valid() || seen[interest] {
			return invalid("interests must be unique predefined choices")
		}
		seen[interest] = true
	}
	birth, err := in.BirthDate.Gregorian()
	if err != nil {
		return invalid("birth_date")
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if birth.After(today) || birth.Before(today.AddDate(-120, 0, 0)) {
		return invalid("birth_date must be within the past 120 years")
	}
	return nil
}
