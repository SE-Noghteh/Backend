package entities

type Appeals struct {
	Id              string  `gorm:"column:id;primaryKey"`
	AppellantUserId string  `gorm:"column:appellant_user_id"`
	TargetType      string  `gorm:"column:target_type"`
	TargetId        string  `gorm:"column:target_id"`
	Reason          *string `gorm:"column:reason"`
	Status          *string `gorm:"column:status"`
	ReviewerUserId  *string `gorm:"column:reviewer_user_id"`
	Result          *string `gorm:"column:result"`
}

func (Appeals) TableName() string { return "appeals" }
