package general

import (
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type IndividualEmployerIdentity struct {
	valueobjects.EmployerReviewMetadata
	SignupIdentity
	EmployerType enums.EmployerType         `json:"employer_type"`
	ReviewStatus enums.EmployerReviewStatus `json:"review_status"`
}

type IndividualEmployerResponse struct {
	Data IndividualEmployerIdentity `json:"data"`
}

type IndividualEmployerLoginResponse struct {
	AccessTokenResponse
	Data IndividualEmployerIdentity `json:"data"`
}

type DocumentURLResponse struct {
	URL       string `json:"url"`
	ExpiresIn int    `json:"expires_in" example:"300"`
}

func identityFromIndividualEmployer(e *entities.IndividualEmployer) IndividualEmployerIdentity {
	return IndividualEmployerIdentity{EmployerReviewMetadata: e.EmployerReviewMetadata, SignupIdentity: SignupIdentity{ID: e.Employer.ID, FirstName: e.FirstName, LastName: e.LastName, Phone: e.Phone, CreatedAt: e.CreatedAt}, EmployerType: e.Employer.Type, ReviewStatus: e.ReviewStatus}
}
