package ports

import (
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type TokenManager interface {
	IssueAccessToken(uint64, enums.AccountRole) (*valueobjects.AccessToken, error)
	VerifyAccessToken(string, enums.AccountRole) (*valueobjects.Principal, error)
}

type JobSeekerTokenManager interface {
	TokenManager
	IssueJobSeekerAccessToken(uint64, uint64) (*valueobjects.AccessToken, error)
}

type ParentTokenManager interface {
	TokenManager
	IssueParentAccessToken(uint64, uint64) (*valueobjects.AccessToken, error)
}

type EmployerTokenManager interface {
	TokenManager
	IssueEmployerAccessToken(uint64, enums.AccountRole, uint64) (*valueobjects.AccessToken, error)
}
