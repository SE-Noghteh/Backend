package postgres

import (
	"fmt"

	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func individualEmployerToModel(e *entities.IndividualEmployer) *models.IndividualEmployerModel {
	return &models.IndividualEmployerModel{
		EmployerID: e.Employer.ID, FirstName: e.FirstName, LastName: e.LastName, FatherName: e.FatherName, NationalCode: e.NationalCode,
		Gender: e.Gender, BirthDate: e.BirthDate.String(), Province: e.Province, City: e.City, Address: e.Address, PostalCode: e.PostalCode,
		BusinessName: e.BusinessName, BusinessActivity: e.BusinessActivity, Website: e.Website, HasLegalGuardian: e.HasLegalGuardian,
		Phone: e.Phone, Email: e.Email, PasswordHash: e.PasswordHash, IdentityDocument: e.IdentityDocument, BusinessLicense: e.BusinessLicense,
		VerificationVideo: e.VerificationVideo, PhoneVerifiedAt: e.PhoneVerifiedAt, CreatedAt: e.CreatedAt, ReviewStatus: e.ReviewStatus,
	}
}

func individualEmployerFromModel(m *models.IndividualEmployerModel) (*entities.IndividualEmployer, error) {
	var birth valueobjects.JalaliDate
	if n, err := fmt.Sscanf(m.BirthDate, "%d/%d/%d", &birth.Year, &birth.Month, &birth.Day); err != nil || n != 3 {
		return nil, fmt.Errorf("invalid persisted birth date")
	}
	return &entities.IndividualEmployer{
		Employer: entities.Employer{ID: m.EmployerID, Type: enums.EmployerIndividual}, FirstName: m.FirstName, LastName: m.LastName, FatherName: m.FatherName, NationalCode: m.NationalCode,
		Gender: m.Gender, BirthDate: birth, Province: m.Province, City: m.City, Address: m.Address, PostalCode: m.PostalCode,
		BusinessName: m.BusinessName, BusinessActivity: m.BusinessActivity, Website: m.Website, HasLegalGuardian: m.HasLegalGuardian,
		Phone: m.Phone, Email: m.Email, PasswordHash: m.PasswordHash, IdentityDocument: m.IdentityDocument, BusinessLicense: m.BusinessLicense,
		VerificationVideo: m.VerificationVideo, PhoneVerifiedAt: m.PhoneVerifiedAt, CreatedAt: m.CreatedAt, ReviewStatus: m.ReviewStatus,
	}, nil
}
