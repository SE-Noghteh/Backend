package enums

import "database/sql/driver"

type ItemStatus uint

const (
	ItemStatusOpen    ItemStatus = iota + 1
	ItemStatusDone    ItemStatus = iota + 1
	ItemStatusExpired ItemStatus = iota + 1
)

func (value ItemStatus) String() string {
	switch value {
	case ItemStatusOpen:
		return "open"
	case ItemStatusDone:
		return "done"
	case ItemStatusExpired:
		return "expired"
	}
	return ""
}

func GetAllItemStatuses() []ItemStatus {
	return []ItemStatus{
		ItemStatusOpen,
		ItemStatusDone,
		ItemStatusExpired,
	}
}

func (value ItemStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ItemStatus) Scan(src any) error {
	return scanEnum(src, map[string]ItemStatus{
		"open":    ItemStatusOpen,
		"done":    ItemStatusDone,
		"expired": ItemStatusExpired,
	}, value)
}
