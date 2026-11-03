package entities

type ActivityCategories struct {
	Id          string  `gorm:"column:id;primaryKey"`
	Name        string  `gorm:"column:name"`
	Description *string `gorm:"column:description"`
	IsActive    bool    `gorm:"column:is_active"`
}

func (ActivityCategories) TableName() string { return "activity_categories" }
