package entities

type PortfolioItems struct {
	Id          string  `gorm:"column:id;primaryKey"`
	JobSeekerId string  `gorm:"column:job_seeker_id"`
	Title       *string `gorm:"column:title"`
	Description *string `gorm:"column:description"`
	FileId      *string `gorm:"column:file_id"`
	Visibility  string  `gorm:"column:visibility"`
}

func (PortfolioItems) TableName() string { return "portfolio_items" }
