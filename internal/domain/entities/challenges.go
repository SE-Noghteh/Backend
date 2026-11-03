package entities

type Challenges struct {
	Id             string  `gorm:"column:id;primaryKey"`
	SkillId        string  `gorm:"column:skill_id"`
	Level          *string `gorm:"column:level"`
	Description    *string `gorm:"column:description"`
	InputFileId    *string `gorm:"column:input_file_id"`
	ExpectedOutput *string `gorm:"column:expected_output"`
	Criteria       *string `gorm:"column:criteria"`
	MaxAttempts    *int32  `gorm:"column:max_attempts"`
}

func (Challenges) TableName() string { return "challenges" }
