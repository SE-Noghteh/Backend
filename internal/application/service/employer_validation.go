package service

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"unicode"
	domainerrors "zedlancer/backend/internal/domain/errors"
)

var postalCodePattern = regexp.MustCompile(`^[0-9]{10}$`)

func validateEmployerContact(password, email, website string) error {
	invalid := func(field string) error { return fmt.Errorf("%w: %s", domainerrors.ErrInvalidSignup, field) }
	if !strongEmployerPassword(password) {
		return invalid("password must contain 12-72 bytes with uppercase, lowercase, a digit, and a symbol")
	}
	if email != "" {
		parsed, err := mail.ParseAddress(email)
		if err != nil || parsed.Address != email {
			return invalid("email")
		}
	}
	if website != "" {
		parsed, err := url.Parse(website)
		if err != nil || parsed.Hostname() == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
			return invalid("website must be an HTTP or HTTPS URL")
		}
	}
	return nil
}

func strongEmployerPassword(password string) bool {
	if len(password) < 12 || len(password) > 72 {
		return false
	}
	var upper, lower, digit, symbol bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			symbol = true
		}
	}
	return upper && lower && digit && symbol
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
