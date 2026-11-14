package enums

import "database/sql/driver"

type CollaborationStatus uint

const (
	CollaborationStatusActive    CollaborationStatus = iota + 1
	CollaborationStatusPaused    CollaborationStatus = iota + 1
	CollaborationStatusCompleted CollaborationStatus = iota + 1
	CollaborationStatusCancelled CollaborationStatus = iota + 1
	CollaborationStatusDisputed  CollaborationStatus = iota + 1
)

func (value CollaborationStatus) String() string {
	switch value {
	case CollaborationStatusActive:
		return "active"
	case CollaborationStatusPaused:
		return "paused"
	case CollaborationStatusCompleted:
		return "completed"
	case CollaborationStatusCancelled:
		return "cancelled"
	case CollaborationStatusDisputed:
		return "disputed"
	}
	return ""
}

func GetAllCollaborationStatuses() []CollaborationStatus {
	return []CollaborationStatus{
		CollaborationStatusActive,
		CollaborationStatusPaused,
		CollaborationStatusCompleted,
		CollaborationStatusCancelled,
		CollaborationStatusDisputed,
	}
}

func (value CollaborationStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *CollaborationStatus) Scan(src any) error {
	return scanEnum(src, map[string]CollaborationStatus{
		"active":    CollaborationStatusActive,
		"paused":    CollaborationStatusPaused,
		"completed": CollaborationStatusCompleted,
		"cancelled": CollaborationStatusCancelled,
		"disputed":  CollaborationStatusDisputed,
	}, value)
}
