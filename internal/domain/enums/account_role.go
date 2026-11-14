package enums

type AccountRole string

const (
	RoleAdmin              AccountRole = "admin"
	RoleJobSeeker          AccountRole = "job_seeker"
	RoleIndividualEmployer AccountRole = "individual_employer"
	RoleLegalEmployer      AccountRole = "legal_employer"
)

func (role AccountRole) Valid() bool {
	return role == RoleJobSeeker || role == RoleIndividualEmployer || role == RoleLegalEmployer || role == RoleAdmin
}
