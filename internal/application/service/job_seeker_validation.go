package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/ports"
)

var mobilePattern = regexp.MustCompile(`^09[0-9]{9}$`)
var codePattern = regexp.MustCompile(`^[0-9]{6}$`)
var nationalPattern = regexp.MustCompile(`^[0-9]{10}$`)

func normalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	if strings.HasPrefix(phone, "+98") {
		phone = "0" + phone[3:]
	} else if strings.HasPrefix(phone, "98") {
		phone = "0" + phone[2:]
	}
	if !mobilePattern.MatchString(phone) {
		return "", fmt.Errorf("%w: phone must be an Iranian mobile number", ports.ErrInvalidSignup)
	}
	return phone, nil
}

func validNationalCode(code string) bool {
	if !nationalPattern.MatchString(code) || code == strings.Repeat(code[:1], 10) {
		return false
	}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(code[i]-'0') * (10 - i)
	}
	check := sum % 11
	if check >= 2 {
		check = 11 - check
	}
	return int(code[9]-'0') == check
}

func validateSignup(in *usecase.SignupJobSeeker) error {
	in.FirstName, in.LastName, in.Address = strings.TrimSpace(in.FirstName), strings.TrimSpace(in.LastName), strings.TrimSpace(in.Address)
	invalid := func(field string) error { return fmt.Errorf("%w: %s", ports.ErrInvalidSignup, field) }
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
	cityOK := false
	for _, city := range entities.Cities() {
		if in.City == city {
			cityOK = true
		}
	}
	if !cityOK {
		return invalid("city")
	}
	if in.Gender != entities.GenderFemale && in.Gender != entities.GenderMale {
		return invalid("gender")
	}
	if len(in.Interests) < 1 || len(in.Interests) > 10 {
		return invalid("interests")
	}
	seen := map[entities.Interest]bool{}
	for _, interest := range in.Interests {
		found := false
		for _, allowed := range entities.Interests() {
			if interest == allowed {
				found = true
			}
		}
		if !found || seen[interest] {
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
