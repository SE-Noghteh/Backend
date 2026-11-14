package enums

import "database/sql/driver"

type AccountStatus uint

const (
	AccountStatusActive                AccountStatus = iota + 1
	AccountStatusPendingAdminApproval  AccountStatus = iota + 1
	AccountStatusRejectedPendingEdit   AccountStatus = iota + 1
	AccountStatusPendingParentApproval AccountStatus = iota + 1
	AccountStatusInactive              AccountStatus = iota + 1
)

func (value AccountStatus) String() string {
	switch value {
	case AccountStatusActive:
		return "active"
	case AccountStatusPendingAdminApproval:
		return "pending_admin_approval"
	case AccountStatusRejectedPendingEdit:
		return "rejected_pending_edit"
	case AccountStatusPendingParentApproval:
		return "pending_parent_approval"
	case AccountStatusInactive:
		return "inactive"
	}
	return ""
}

func GetAllAccountStatuses() []AccountStatus {
	return []AccountStatus{
		AccountStatusActive,
		AccountStatusPendingAdminApproval,
		AccountStatusRejectedPendingEdit,
		AccountStatusPendingParentApproval,
		AccountStatusInactive,
	}
}

func (value AccountStatus) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *AccountStatus) Scan(src any) error {
	return scanEnum(src, map[string]AccountStatus{
		"active":                  AccountStatusActive,
		"pending_admin_approval":  AccountStatusPendingAdminApproval,
		"rejected_pending_edit":   AccountStatusRejectedPendingEdit,
		"pending_parent_approval": AccountStatusPendingParentApproval,
		"inactive":                AccountStatusInactive,
	}, value)
}
