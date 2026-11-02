package ports

import "zedlancer/backend/internal/domain/valueobjects"

type TokenManager interface {
	IssueAccessToken(uint64) (*valueobjects.AccessToken, error)
	VerifyAccessToken(string) (*valueobjects.JobSeekerPrincipal, error)
}
