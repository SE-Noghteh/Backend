package models

// Password reset challenges are isolated from signup and phone-change OTPs.
type JobSeekerPasswordResetOTPModel JobSeekerPhoneChangeOTPModel

func (JobSeekerPasswordResetOTPModel) TableName() string { return "job_seeker_password_reset_otps" }
