package entities

type ExtensionRequests struct {
	Id                   string `gorm:"column:id;primaryKey"`
	CollaborationId      string `gorm:"column:collaboration_id"`
	NewContractVersionId string `gorm:"column:new_contract_version_id"`
}

func (ExtensionRequests) TableName() string { return "extension_requests" }
