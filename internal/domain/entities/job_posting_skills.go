package entities

type JobPostingSkills struct {
	JobPostingId string `gorm:"column:job_posting_id;primaryKey"`
	SkillId      string `gorm:"column:skill_id;primaryKey"`
}

func (JobPostingSkills) TableName() string { return "job_posting_skills" }
