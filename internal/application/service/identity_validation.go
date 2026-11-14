package service

import (
	"fmt"
	"regexp"
	"strings"

	domainerrors "zedlancer/backend/internal/domain/errors"
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
		return "", fmt.Errorf("%w: phone must be an Iranian mobile number", domainerrors.ErrInvalidSignup)
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
