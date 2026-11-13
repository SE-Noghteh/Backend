package entities

type DeliverableFiles struct {
	DeliverableId string `gorm:"column:deliverable_id;primaryKey"`
	FileId        string `gorm:"column:file_id;primaryKey"`
}

func (DeliverableFiles) TableName() string { return "deliverable_files" }
