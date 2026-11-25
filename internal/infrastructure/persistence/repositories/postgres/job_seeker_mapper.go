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
		SessionVersion:     seeker.SessionVersion,
		AdminReviewModel:   adminReviewToModel(seeker.AdminReview),
		ParentID:           seeker.ParentID,
		ParentRelationship: seeker.ParentRelationship,
		ID:                 seeker.ID,
		FirstName:          seeker.FirstName,
		LastName:           seeker.LastName,
		NationalCode:       seeker.NationalCode,
		Province:           string(seeker.Province),
		City:               string(seeker.City),
		Address:            seeker.Address,
		PostalCode:         seeker.PostalCode,
		BirthDate:          seeker.BirthDate.String(),
		Gender:             string(seeker.Gender),
		Interests:          seeker.Interests,
		Phone:              seeker.Phone,
		PasswordHash:       seeker.PasswordHash,
		PhoneVerifiedAt:    seeker.PhoneVerifiedAt,
		CreatedAt:          seeker.CreatedAt,
		IdentityDocument:   seeker.IdentityDocument,
		VerificationVideo:  seeker.VerificationVideo,
	}
}

func jobSeekerFromModel(model *models.JobSeekerModel) (*entities.JobSeeker, error) {
	var birth valueobjects.JalaliDate
	if n, err := fmt.Sscanf(model.BirthDate, "%d/%d/%d", &birth.Year, &birth.Month, &birth.Day); err != nil || n != 3 {
		return nil, fmt.Errorf("invalid persisted birth date")
	}
	return &entities.JobSeeker{
		SessionVersion:            model.SessionVersion,
		ParentAdminApprovalStatus: model.ParentAdminApprovalStatus,
		AdminReview:               adminReviewFromModel(model.AdminReviewModel),
		ParentID:                  model.ParentID,
		ParentRelationship:        model.ParentRelationship,
		ID:                        model.ID,
		FirstName:                 model.FirstName,
		LastName:                  model.LastName,
		NationalCode:              model.NationalCode,
		Province:                  enums.Province(model.Province),
		City:                      enums.City(model.City),
		Address:                   model.Address,
		PostalCode:                model.PostalCode,
		BirthDate:                 birth,
		Gender:                    enums.Gender(model.Gender),
		Interests:                 model.Interests,
		Phone:                     model.Phone,
		PasswordHash:              model.PasswordHash,
		PhoneVerifiedAt:           model.PhoneVerifiedAt,
		CreatedAt:                 model.CreatedAt,
		IdentityDocument:          model.IdentityDocument,
		VerificationVideo:         model.VerificationVideo,
	}, nil
}
