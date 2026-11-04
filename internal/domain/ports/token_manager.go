package ports

import (
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type TokenManager interface {
	IssueAccessToken(uint64, enums.AccountRole) (*valueobjects.AccessToken, error)
	VerifyAccessToken(string, enums.AccountRole) (*valueobjects.Principal, error)
}
