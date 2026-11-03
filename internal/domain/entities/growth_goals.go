package entities

type GrowthGoals struct {
	Id          string `gorm:"column:id;primaryKey"`
	JobSeekerId string `gorm:"column:job_seeker_id"`
	SkillId     string `gorm:"column:skill_id"`
}

func (GrowthGoals) TableName() string { return "growth_goals" }
