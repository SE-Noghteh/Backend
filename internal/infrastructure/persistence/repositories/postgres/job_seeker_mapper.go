package postgres

import (
	"fmt"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func jobSeekerToModel(seeker *entities.JobSeeker) *models.JobSeekerModel {
	return &models.JobSeekerModel{
		SignupStatus:       seeker.SignupStatus,
		ParentID:           seeker.ParentID,
		ParentRelationship: seeker.ParentRelationship,
		ID:                 seeker.ID,
		FirstName:          seeker.FirstName,
		LastName:           seeker.LastName,
		NationalCode:       seeker.NationalCode,
		City:               string(seeker.City),
		Address:            seeker.Address,
		BirthDate:          seeker.BirthDate.String(),
		Gender:             string(seeker.Gender),
		Interests:          seeker.Interests,
		Phone:              seeker.Phone,
		PasswordHash:       seeker.PasswordHash,
		PhoneVerifiedAt:    seeker.PhoneVerifiedAt,
		CreatedAt:          seeker.CreatedAt,
		VerificationVideo:  seeker.VerificationVideo,
	}
}

func jobSeekerFromModel(model *models.JobSeekerModel) (*entities.JobSeeker, error) {
	var birth valueobjects.JalaliDate
	if n, err := fmt.Sscanf(model.BirthDate, "%d/%d/%d", &birth.Year, &birth.Month, &birth.Day); err != nil || n != 3 {
		return nil, fmt.Errorf("invalid persisted birth date")
	}
	return &entities.JobSeeker{
		SignupStatus:       model.SignupStatus,
		ParentID:           model.ParentID,
		ParentRelationship: model.ParentRelationship,
		ID:                 model.ID,
		FirstName:          model.FirstName,
		LastName:           model.LastName,
		NationalCode:       model.NationalCode,
		City:               enums.City(model.City),
		Address:            model.Address,
		BirthDate:          birth,
		Gender:             enums.Gender(model.Gender),
		Interests:          model.Interests,
		Phone:              model.Phone,
		PasswordHash:       model.PasswordHash,
		PhoneVerifiedAt:    model.PhoneVerifiedAt,
		CreatedAt:          model.CreatedAt,
		VerificationVideo:  model.VerificationVideo,
	}, nil
}
