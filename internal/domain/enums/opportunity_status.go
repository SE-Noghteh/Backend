package enums

import "database/sql/driver"

type OpportunityStatus uint

const (
	OpportunityStatusDraft     OpportunityStatus = iota + 1
	OpportunityStatusInReview  OpportunityStatus = iota + 1
	OpportunityStatusPublished OpportunityStatus = iota + 1
	OpportunityStatusClosed    OpportunityStatus = iota + 1
	OpportunityStatusArchived  OpportunityStatus = iota + 1
)

func (value OpportunityStatus) String() string {
	switch value {
	case OpportunityStatusDraft:
		return "draft"
	case OpportunityStatusInReview:
		return "in_review"
	case OpportunityStatusPublished:
		return "published"
	case OpportunityStatusClosed:
		return "closed"
	case OpportunityStatusArchived:
		return "archived"
	}
	return ""
}

func GetAllOpportunityStatuses() []OpportunityStatus {
	return []OpportunityStatus{
		OpportunityStatusDraft,
		OpportunityStatusInReview,
		OpportunityStatusPublished,
		OpportunityStatusClosed,
		OpportunityStatusArchived,
	}
}

func (value OpportunityStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *OpportunityStatus) Scan(src any) error {
	return scanEnum(src, map[string]OpportunityStatus{
		"draft":     OpportunityStatusDraft,
		"in_review": OpportunityStatusInReview,
		"published": OpportunityStatusPublished,
		"closed":    OpportunityStatusClosed,
		"archived":  OpportunityStatusArchived,
	}, value)
}
