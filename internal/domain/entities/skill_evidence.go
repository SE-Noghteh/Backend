package entities

type SkillEvidence struct {
	Id                 string  `gorm:"column:id;primaryKey"`
	JobSeekerId        string  `gorm:"column:job_seeker_id"`
	SkillId            string  `gorm:"column:skill_id"`
	CollaborationId    *string `gorm:"column:collaboration_id"`
	PortfolioItemId    *string `gorm:"column:portfolio_item_id"`
	ChallengeAttemptId *string `gorm:"column:challenge_attempt_id"`
	MentorSummary      *string `gorm:"column:mentor_summary"`
}

func (SkillEvidence) TableName() string { return "skill_evidence" }
