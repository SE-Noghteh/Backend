package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

var companyNationalIDPattern = regexp.MustCompile(`^[0-9]{11}$`)
var registrationNumberPattern = regexp.MustCompile(`^[0-9]{1,30}$`)
var economicCodePattern = regexp.MustCompile(`^[0-9]{12}$`)

func validateLegalEmployer(in *usecase.SignupLegalEmployer) error {
	invalid := func(field string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidSignup, field) }
	for _, field := range []struct {
		name     string
		value    *string
		max      int
		required bool
	}{
		{"company_name", &in.CompanyName, 200, true},
		{"representative_first_name", &in.RepresentativeFirstName, 100, true},
		{"representative_last_name", &in.RepresentativeLastName, 100, true},
		{"address", &in.Address, 1000, true},
		{"website", &in.Website, 2048, false}, {"email", &in.Email, 254, false},
	} {
		*field.value = strings.TrimSpace(*field.value)
		if (field.required && *field.value == "") || utf8.RuneCountInString(*field.value) > field.max {
			return invalid(field.name)
		}
	}
	if !in.LegalEntityType.Valid() {
		return invalid("legal_entity_type")
	}
	if !companyNationalIDPattern.MatchString(in.CompanyNationalID) || strings.Trim(in.CompanyNationalID, in.CompanyNationalID[:1]) == "" {
		return invalid("company_national_id must contain eleven digits")
	}
	if !registrationNumberPattern.MatchString(in.RegistrationNumber) {
		return invalid("registration_number")
	}
	if !economicCodePattern.MatchString(in.EconomicCode) {
		return invalid("economic_code must contain twelve digits")
	}
	if !in.Province.Valid() || !in.City.Valid() || in.City.Province() != in.Province {
		return invalid("province and city must match")
	}
	if !postalCodePattern.MatchString(in.PostalCode) {
		return invalid("postal_code must contain ten digits")
	}
	if !in.RepresentativePosition.Valid() {
		return invalid("representative_position")
	}
	if !in.RepresentativeGender.Valid() {
		return invalid("representative_gender")
	}
	if !validNationalCode(in.RepresentativeNationalCode) {
		return invalid("representative_national_code")
	}
	if len(in.Activities) == 0 || len(in.Activities) > len(enums.Interests()) {
		return invalid("select 1-10 activities from signup-options")
	}
	seen := make(map[enums.Interest]bool)
	for _, activity := range in.Activities {
		if !activity.Valid() || seen[activity] {
			return invalid("activities must be distinct choices from signup-options")
		}
		seen[activity] = true
	}
	if in.RequiresActivityLicense && in.ActivityLicense == nil {
		return invalid("activity_license is required for licensed activities")
	}
	if in.RepresentativePosition == enums.RepresentativeAttorney && in.AuthorizationDocument == nil {
		return invalid("authorization_document is required for an attorney")
	}
	return validateEmployerContact(in.Password, in.Email, in.Website)
}
