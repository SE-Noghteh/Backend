package entities

type ChallengeAttempts struct {
	Id           string  `gorm:"column:id;primaryKey"`
	ChallengeId  string  `gorm:"column:challenge_id"`
	JobSeekerId  string  `gorm:"column:job_seeker_id"`
	AnswerText   *string `gorm:"column:answer_text"`
	AnswerFileId *string `gorm:"column:answer_file_id"`
	EvalMethod   *string `gorm:"column:eval_method"`
	Result       *string `gorm:"column:result"`
	Feedback     *string `gorm:"column:feedback"`
}

func (ChallengeAttempts) TableName() string { return "challenge_attempts" }
