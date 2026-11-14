package enums

import "database/sql/driver"

type FinanceStatus uint

const (
	FinanceStatusNone        FinanceStatus = iota + 1
	FinanceStatusPending     FinanceStatus = iota + 1
	FinanceStatusSuccess     FinanceStatus = iota + 1
	FinanceStatusFailed      FinanceStatus = iota + 1
	FinanceStatusRefunded    FinanceStatus = iota + 1
	FinanceStatusNeedsReview FinanceStatus = iota + 1
)

func (value FinanceStatus) String() string {
	switch value {
	case FinanceStatusNone:
		return "none"
	case FinanceStatusPending:
		return "pending"
	case FinanceStatusSuccess:
		return "success"
	case FinanceStatusFailed:
		return "failed"
	case FinanceStatusRefunded:
		return "refunded"
	case FinanceStatusNeedsReview:
		return "needs_review"
	}
	return ""
}

func GetAllFinanceStatuses() []FinanceStatus {
	return []FinanceStatus{
		FinanceStatusNone,
		FinanceStatusPending,
		FinanceStatusSuccess,
		FinanceStatusFailed,
		FinanceStatusRefunded,
		FinanceStatusNeedsReview,
	}
}

func (value FinanceStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *FinanceStatus) Scan(src any) error {
	return scanEnum(src, map[string]FinanceStatus{
		"none":         FinanceStatusNone,
		"pending":      FinanceStatusPending,
		"success":      FinanceStatusSuccess,
		"failed":       FinanceStatusFailed,
		"refunded":     FinanceStatusRefunded,
		"needs_review": FinanceStatusNeedsReview,
	}, value)
}
