package entities

type Categories struct {
	Id       string  `gorm:"column:id;primaryKey"`
	ParentId *string `gorm:"column:parent_id"`
	Title    string  `gorm:"column:title"`
	Active   *bool   `gorm:"column:active"`
}

func (Categories) TableName() string { return "categories" }
