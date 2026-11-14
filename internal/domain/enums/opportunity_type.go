package enums

import "database/sql/driver"

type OpportunityType uint

const (
	OpportunityTypeWork       OpportunityType = iota + 1
	OpportunityTypeInternship OpportunityType = iota + 1
)

func (value OpportunityType) String() string {
	switch value {
	case OpportunityTypeWork:
		return "work"
	case OpportunityTypeInternship:
		return "internship"
	}
	return ""
}

func GetAllOpportunityTypes() []OpportunityType {
	return []OpportunityType{
		OpportunityTypeWork,
		OpportunityTypeInternship,
	}
}

func (value OpportunityType) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *OpportunityType) Scan(src any) error {
	return scanEnum(src, map[string]OpportunityType{
		"work":       OpportunityTypeWork,
		"internship": OpportunityTypeInternship,
	}, value)
}
