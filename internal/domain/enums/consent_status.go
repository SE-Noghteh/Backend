package enums

import "database/sql/driver"

type ConsentStatus uint

const (
	ConsentStatusPending   ConsentStatus = iota + 1
	ConsentStatusGranted   ConsentStatus = iota + 1
	ConsentStatusDenied    ConsentStatus = iota + 1
	ConsentStatusWithdrawn ConsentStatus = iota + 1
	ConsentStatusExpired   ConsentStatus = iota + 1
)

func (value ConsentStatus) String() string {
	switch value {
	case ConsentStatusPending:
		return "pending"
	case ConsentStatusGranted:
		return "granted"
	case ConsentStatusDenied:
		return "denied"
	case ConsentStatusWithdrawn:
		return "withdrawn"
	case ConsentStatusExpired:
		return "expired"
	}
	return ""
}

func GetAllConsentStatuses() []ConsentStatus {
	return []ConsentStatus{
		ConsentStatusPending,
		ConsentStatusGranted,
		ConsentStatusDenied,
		ConsentStatusWithdrawn,
		ConsentStatusExpired,
	}
}

func (value ConsentStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ConsentStatus) Scan(src any) error {
	return scanEnum(src, map[string]ConsentStatus{
		"pending":   ConsentStatusPending,
		"granted":   ConsentStatusGranted,
		"denied":    ConsentStatusDenied,
		"withdrawn": ConsentStatusWithdrawn,
		"expired":   ConsentStatusExpired,
	}, value)
}
