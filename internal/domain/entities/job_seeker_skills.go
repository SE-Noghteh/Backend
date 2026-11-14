package entities

type JobSeekerSkills struct {
	JobSeekerId string `gorm:"column:job_seeker_id;primaryKey"`
	SkillId     string `gorm:"column:skill_id;primaryKey"`
}

func (JobSeekerSkills) TableName() string { return "job_seeker_skills" }
