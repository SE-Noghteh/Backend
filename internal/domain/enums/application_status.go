package enums

import "database/sql/driver"

type ApplicationStatus uint

const (
	ApplicationStatusSent        ApplicationStatus = iota + 1
	ApplicationStatusWithdrawn   ApplicationStatus = iota + 1
	ApplicationStatusShortlisted ApplicationStatus = iota + 1
	ApplicationStatusRejected    ApplicationStatus = iota + 1
	ApplicationStatusOffered     ApplicationStatus = iota + 1
	ApplicationStatusAccepted    ApplicationStatus = iota + 1
)

func (value ApplicationStatus) String() string {
	switch value {
	case ApplicationStatusSent:
		return "sent"
	case ApplicationStatusWithdrawn:
		return "withdrawn"
	case ApplicationStatusShortlisted:
		return "shortlisted"
	case ApplicationStatusRejected:
		return "rejected"
	case ApplicationStatusOffered:
		return "offered"
	case ApplicationStatusAccepted:
		return "accepted"
	}
	return ""
}

func GetAllApplicationStatuses() []ApplicationStatus {
	return []ApplicationStatus{
		ApplicationStatusSent,
		ApplicationStatusWithdrawn,
		ApplicationStatusShortlisted,
		ApplicationStatusRejected,
		ApplicationStatusOffered,
		ApplicationStatusAccepted,
	}
}

func (value ApplicationStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ApplicationStatus) Scan(src any) error {
	return scanEnum(src, map[string]ApplicationStatus{
		"sent":        ApplicationStatusSent,
		"withdrawn":   ApplicationStatusWithdrawn,
		"shortlisted": ApplicationStatusShortlisted,
		"rejected":    ApplicationStatusRejected,
		"offered":     ApplicationStatusOffered,
		"accepted":    ApplicationStatusAccepted,
	}, value)
}
