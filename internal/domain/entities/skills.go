package entities

type Skills struct {
	Id          string  `gorm:"column:id;primaryKey"`
	Name        string  `gorm:"column:name"`
	Description *string `gorm:"column:description"`
	IsActive    bool    `gorm:"column:is_active"`
}

func (Skills) TableName() string { return "skills" }
