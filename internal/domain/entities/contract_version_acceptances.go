package entities

import (
	"time"
)

type ContractVersionAcceptances struct {
	Id                string    `gorm:"column:id;primaryKey"`
	ContractVersionId string    `gorm:"column:contract_version_id"`
	AcceptorUserId    string    `gorm:"column:acceptor_user_id"`
	AcceptedAt        time.Time `gorm:"column:accepted_at"`
}

func (ContractVersionAcceptances) TableName() string { return "contract_version_acceptances" }
