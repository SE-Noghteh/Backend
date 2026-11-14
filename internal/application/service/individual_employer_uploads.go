package service

import (
	"context"
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/entities"
	"zedlancer/backend/internal/domain/valueobjects"
)

type employerDocuments struct {
	identity, video valueobjects.StoredObject
	license         *valueobjects.StoredObject
}

func employerObjectKeys(employer *entities.IndividualEmployer) []string {
	if employer == nil {
		return nil
	}
	keys := []string{employer.IdentityDocument.Key, employer.VerificationVideo.Key}
	if employer.BusinessLicense != nil {
		keys = append(keys, employer.BusinessLicense.Key)
	}
	return keys
}

func (s *IndividualEmployerService) uploadSignupDocuments(ctx context.Context, in usecase.SignupIndividualEmployer, uploaded *[]string) (*employerDocuments, error) {
	objects, err := uploadEmployerFiles(ctx, s.storage, s.media, "individual-employers", []employerUpload{
		{kind: "identity", file: &in.IdentityDocument}, {kind: "verification", file: &in.VerificationVideo, video: true}, {kind: "business-license", file: in.BusinessLicense},
	}, uploaded)
	if err != nil {
		return nil, err
	}
	return &employerDocuments{identity: *objects["identity"], video: *objects["verification"], license: objects["business-license"]}, nil
}
