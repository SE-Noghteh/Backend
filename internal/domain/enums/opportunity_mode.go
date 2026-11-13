package enums

import "database/sql/driver"

type OpportunityMode uint

const (
	OpportunityModeOnline OpportunityMode = iota + 1
	OpportunityModeOnsite OpportunityMode = iota + 1
)

func (value OpportunityMode) String() string {
	switch value {
	case OpportunityModeOnline:
		return "online"
	case OpportunityModeOnsite:
		return "onsite"
	}
	return ""
}

func GetAllOpportunityModes() []OpportunityMode {
	return []OpportunityMode{
		OpportunityModeOnline,
		OpportunityModeOnsite,
	}
}

func (value OpportunityMode) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *OpportunityMode) Scan(src any) error {
	return scanEnum(src, map[string]OpportunityMode{
		"online": OpportunityModeOnline,
		"onsite": OpportunityModeOnsite,
	}, value)
}
