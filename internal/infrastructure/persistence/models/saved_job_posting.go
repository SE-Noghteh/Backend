package models

type SavedJobPostingModel struct {
	JobSeekerID  uint64          `gorm:"primaryKey;autoIncrement:false"`
	JobPostingID uint64          `gorm:"primaryKey;autoIncrement:false;index"`
	JobSeeker    JobSeekerModel  `gorm:"foreignKey:JobSeekerID;constraint:OnDelete:CASCADE"`
	JobPosting   JobPostingModel `gorm:"foreignKey:JobPostingID;constraint:OnDelete:CASCADE"`
}

func (SavedJobPostingModel) TableName() string { return "saved_job_postings" }
