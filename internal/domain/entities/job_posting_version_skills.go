package entities

type JobPostingVersionSkills struct {
	JobPostingVersionId string `gorm:"column:job_posting_version_id;primaryKey"`
	SkillId             string `gorm:"column:skill_id;primaryKey"`
}

func (JobPostingVersionSkills) TableName() string { return "job_posting_version_skills" }
