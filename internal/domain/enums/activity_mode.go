package enums

import "database/sql/driver"

type ActivityMode uint

const (
	ActivityModeOnline ActivityMode = iota + 1
	ActivityModeOnsite ActivityMode = iota + 1
	ActivityModeBoth   ActivityMode = iota + 1
)

func (value ActivityMode) String() string {
	switch value {
	case ActivityModeOnline:
		return "online"
	case ActivityModeOnsite:
		return "onsite"
	case ActivityModeBoth:
		return "both"
	}
	return ""
}

func GetAllActivityModes() []ActivityMode {
	return []ActivityMode{
		ActivityModeOnline,
		ActivityModeOnsite,
		ActivityModeBoth,
	}
}

func (value ActivityMode) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ActivityMode) Scan(src any) error {
	return scanEnum(src, map[string]ActivityMode{
		"online": ActivityModeOnline,
		"onsite": ActivityModeOnsite,
		"both":   ActivityModeBoth,
	}, value)
}
