package entities

type SavedJobPostings struct {
	JobSeekerId  string `gorm:"column:job_seeker_id;primaryKey"`
	JobPostingId string `gorm:"column:job_posting_id;primaryKey"`
}

func (SavedJobPostings) TableName() string { return "saved_job_postings" }
