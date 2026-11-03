package entities

type Activities struct {
	Id                      string  `gorm:"column:id;primaryKey"`
	CategoryId              string  `gorm:"column:category_id"`
	Title                   string  `gorm:"column:title;unique"`
	MinRequirements         *string `gorm:"column:min_requirements"`
	SupervisionRequirements *string `gorm:"column:supervision_requirements"`
	Active                  *bool   `gorm:"column:active"`
}

func (Activities) TableName() string { return "activities" }
