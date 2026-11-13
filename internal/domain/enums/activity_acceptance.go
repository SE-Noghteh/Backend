package enums

import "database/sql/driver"

type ActivityAcceptance uint

const (
	ActivityAcceptanceAllowedAfterReview ActivityAcceptance = iota + 1
	ActivityAcceptanceSuspended          ActivityAcceptance = iota + 1
	ActivityAcceptanceOutOfScope         ActivityAcceptance = iota + 1
)

func (value ActivityAcceptance) String() string {
	switch value {
	case ActivityAcceptanceAllowedAfterReview:
		return "allowed_after_review"
	case ActivityAcceptanceSuspended:
		return "suspended"
	case ActivityAcceptanceOutOfScope:
		return "out_of_scope"
	}
	return ""
}

func GetAllActivityAcceptances() []ActivityAcceptance {
	return []ActivityAcceptance{
		ActivityAcceptanceAllowedAfterReview,
		ActivityAcceptanceSuspended,
		ActivityAcceptanceOutOfScope,
	}
}

func (value ActivityAcceptance) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ActivityAcceptance) Scan(src any) error {
	return scanEnum(src, map[string]ActivityAcceptance{
		"allowed_after_review": ActivityAcceptanceAllowedAfterReview,
		"suspended":            ActivityAcceptanceSuspended,
		"out_of_scope":         ActivityAcceptanceOutOfScope,
	}, value)
}
