package enums

import "database/sql/driver"

type ReviewStatus uint

const (
	ReviewStatusHidden    ReviewStatus = iota + 1
	ReviewStatusPublished ReviewStatus = iota + 1
	ReviewStatusRemoved   ReviewStatus = iota + 1
)

func (value ReviewStatus) String() string {
	switch value {
	case ReviewStatusHidden:
		return "hidden"
	case ReviewStatusPublished:
		return "published"
	case ReviewStatusRemoved:
		return "removed"
	}
	return ""
}

func GetAllReviewStatuses() []ReviewStatus {
	return []ReviewStatus{
		ReviewStatusHidden,
		ReviewStatusPublished,
		ReviewStatusRemoved,
	}
}

func (value ReviewStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ReviewStatus) Scan(src any) error {
	return scanEnum(src, map[string]ReviewStatus{
		"hidden":    ReviewStatusHidden,
		"published": ReviewStatusPublished,
		"removed":   ReviewStatusRemoved,
	}, value)
}
