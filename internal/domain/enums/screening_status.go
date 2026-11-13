package enums

import "database/sql/driver"

type ScreeningStatus uint

const (
	ScreeningStatusPending  ScreeningStatus = iota + 1
	ScreeningStatusApproved ScreeningStatus = iota + 1
	ScreeningStatusRejected ScreeningStatus = iota + 1
	ScreeningStatusReturned ScreeningStatus = iota + 1
)

func (value ScreeningStatus) String() string {
	switch value {
	case ScreeningStatusPending:
		return "pending"
	case ScreeningStatusApproved:
		return "approved"
	case ScreeningStatusRejected:
		return "rejected"
	case ScreeningStatusReturned:
		return "returned"
	}
	return ""
}

func GetAllScreeningStatuses() []ScreeningStatus {
	return []ScreeningStatus{
		ScreeningStatusPending,
		ScreeningStatusApproved,
		ScreeningStatusRejected,
		ScreeningStatusReturned,
	}
}

func (value ScreeningStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ScreeningStatus) Scan(src any) error {
	return scanEnum(src, map[string]ScreeningStatus{
		"pending":  ScreeningStatusPending,
		"approved": ScreeningStatusApproved,
		"rejected": ScreeningStatusRejected,
		"returned": ScreeningStatusReturned,
	}, value)
}
