package postgres

import (
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/infrastructure/persistence/models"
)

func legalEmployerToModel(e *entities.LegalEmployer) *models.LegalEmployerModel {
	return &models.LegalEmployerModel{EmployerID: e.Employer.ID,
		CompanyName:                e.CompanyName,
		LegalEntityType:            e.LegalEntityType,
		CompanyNationalID:          e.CompanyNationalID,
		RegistrationNumber:         e.RegistrationNumber,
		EconomicCode:               e.EconomicCode,
		Province:                   e.Province,
		City:                       e.City,
		Address:                    e.Address,
		PostalCode:                 e.PostalCode,
		RepresentativeFirstName:    e.RepresentativeFirstName,
		RepresentativeLastName:     e.RepresentativeLastName,
		RepresentativePosition:     e.RepresentativePosition,
		RepresentativeGender:       e.RepresentativeGender,
		RepresentativeNationalCode: e.RepresentativeNationalCode,
		HasTaxFile:                 e.HasTaxFile,
		Activities:                 e.Activities,
		Website:                    e.Website,
		RequiresActivityLicense:    e.RequiresActivityLicense,
		Phone:                      e.Phone,
		Email:                      e.Email,
		PasswordHash:               e.PasswordHash,
		EstablishmentDocument:      e.EstablishmentDocument,
		IdentityDocument:           e.IdentityDocument,
		ActivityLicense:            e.ActivityLicense,
		AuthorizationDocument:      e.AuthorizationDocument,
		VerificationVideo:          e.VerificationVideo,
		PhoneVerifiedAt:            e.PhoneVerifiedAt,
		CreatedAt:                  e.CreatedAt,
		ReviewStatus:               e.ReviewStatus,
	}
}
func legalEmployerFromModel(m *models.LegalEmployerModel) *entities.LegalEmployer {
	return &entities.LegalEmployer{Employer: entities.Employer{ID: m.EmployerID, Type: enums.EmployerLegal},
		CompanyName:                m.CompanyName,
		LegalEntityType:            m.LegalEntityType,
		CompanyNationalID:          m.CompanyNationalID,
		RegistrationNumber:         m.RegistrationNumber,
		EconomicCode:               m.EconomicCode,
		Province:                   m.Province,
		City:                       m.City,
		Address:                    m.Address,
		PostalCode:                 m.PostalCode,
		RepresentativeFirstName:    m.RepresentativeFirstName,
		RepresentativeLastName:     m.RepresentativeLastName,
		RepresentativePosition:     m.RepresentativePosition,
		RepresentativeGender:       m.RepresentativeGender,
		RepresentativeNationalCode: m.RepresentativeNationalCode,
		HasTaxFile:                 m.HasTaxFile,
		Activities:                 m.Activities,
		Website:                    m.Website,
		RequiresActivityLicense:    m.RequiresActivityLicense,
		Phone:                      m.Phone,
		Email:                      m.Email,
		PasswordHash:               m.PasswordHash,
		EstablishmentDocument:      m.EstablishmentDocument,
		IdentityDocument:           m.IdentityDocument,
		ActivityLicense:            m.ActivityLicense,
		AuthorizationDocument:      m.AuthorizationDocument,
		VerificationVideo:          m.VerificationVideo,
		PhoneVerifiedAt:            m.PhoneVerifiedAt,
		CreatedAt:                  m.CreatedAt,
		ReviewStatus:               m.ReviewStatus,
	}
}
