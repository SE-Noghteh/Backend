package enums

import "database/sql/driver"

type DeliverableStatus uint

const (
	DeliverableStatusSubmitted         DeliverableStatus = iota + 1
	DeliverableStatusRevisionRequested DeliverableStatus = iota + 1
	DeliverableStatusApproved          DeliverableStatus = iota + 1
)

func (value DeliverableStatus) String() string {
	switch value {
	case DeliverableStatusSubmitted:
		return "submitted"
	case DeliverableStatusRevisionRequested:
		return "revision_requested"
	case DeliverableStatusApproved:
		return "approved"
	}
	return ""
}

func GetAllDeliverableStatuses() []DeliverableStatus {
	return []DeliverableStatus{
		DeliverableStatusSubmitted,
		DeliverableStatusRevisionRequested,
		DeliverableStatusApproved,
	}
}

func (value DeliverableStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *DeliverableStatus) Scan(src any) error {
	return scanEnum(src, map[string]DeliverableStatus{
		"submitted":          DeliverableStatusSubmitted,
		"revision_requested": DeliverableStatusRevisionRequested,
		"approved":           DeliverableStatusApproved,
	}, value)
}
