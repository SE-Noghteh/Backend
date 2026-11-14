package service

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

func validateIndividualEmployer(in *usecase.SignupIndividualEmployer) error {
	invalid := func(field string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidSignup, field) }
	for _, field := range []struct {
		name     string
		value    *string
		max      int
		required bool
	}{
		{"first_name", &in.FirstName, 100, true}, {"last_name", &in.LastName, 100, true}, {"father_name", &in.FatherName, 100, true},
		{"address", &in.Address, 1000, true}, {"business_name", &in.BusinessName, 200, false}, {"business_activity", &in.BusinessActivity, 1000, false},
		{"website", &in.Website, 2048, false}, {"email", &in.Email, 254, false},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if (field.required && *field.value == "") || utf8.RuneCountInString(*field.value) > field.max {
			return invalid(field.name)
		}
	}
	if !validNationalCode(in.NationalCode) {
		return invalid("national_code")
	}
	if !in.Gender.Valid() {
		return invalid("gender")
	}
	if !in.Province.Valid() || !in.City.Valid() || in.City.Province() != in.Province {
		return invalid("province and city must match")
	}
	if !postalCodePattern.MatchString(in.PostalCode) {
		return invalid("postal_code must contain ten digits")
	}
	birth, err := in.BirthDate.Gregorian()
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if err != nil || birth.After(today) || birth.Before(today.AddDate(-120, 0, 0)) {
		return invalid("birth_date")
	}
	return validateEmployerContact(in.Password, in.Email, in.Website)
}
