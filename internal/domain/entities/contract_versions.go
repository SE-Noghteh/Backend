package entities

import (
	"encoding/json"
	"time"
)

type ContractVersions struct {
	Id              string          `gorm:"column:id;primaryKey"`
	WorkContractId  string          `gorm:"column:work_contract_id"`
	VersionNo       int32           `gorm:"column:version_no"`
	Terms           json.RawMessage `gorm:"column:terms"`
	CreatedByUserId *string         `gorm:"column:created_by_user_id"`
	CreatedAt       *time.Time      `gorm:"column:created_at"`
}

func (ContractVersions) TableName() string { return "contract_versions" }
