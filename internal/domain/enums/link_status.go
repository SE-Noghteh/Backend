package enums

import "database/sql/driver"

type LinkStatus uint

const (
	LinkStatusInvited LinkStatus = iota + 1
	LinkStatusActive  LinkStatus = iota + 1
	LinkStatusRevoked LinkStatus = iota + 1
)

func (value LinkStatus) String() string {
	switch value {
	case LinkStatusInvited:
		return "invited"
	case LinkStatusActive:
		return "active"
	case LinkStatusRevoked:
		return "revoked"
	}
	return ""
}

func GetAllLinkStatuses() []LinkStatus {
	return []LinkStatus{
		LinkStatusInvited,
		LinkStatusActive,
		LinkStatusRevoked,
	}
}

func (value LinkStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *LinkStatus) Scan(src any) error {
	return scanEnum(src, map[string]LinkStatus{
		"invited": LinkStatusInvited,
		"active":  LinkStatusActive,
		"revoked": LinkStatusRevoked,
	}, value)
}
