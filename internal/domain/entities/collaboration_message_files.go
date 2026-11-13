package entities

type CollaborationMessageFiles struct {
	CollaborationMessageId string `gorm:"column:collaboration_message_id;primaryKey"`
	FileId                 string `gorm:"column:file_id;primaryKey"`
}

func (CollaborationMessageFiles) TableName() string { return "collaboration_message_files" }
