package enums

type AccountRole string

const (
	RoleParent             AccountRole = "parent"
	RoleAdmin              AccountRole = "admin"
	RoleJobSeeker          AccountRole = "job_seeker"
	RoleIndividualEmployer AccountRole = "individual_employer"
	RoleLegalEmployer      AccountRole = "legal_employer"
)

func (role AccountRole) Valid() bool {
	return role == RoleParent || role == RoleJobSeeker || role == RoleIndividualEmployer || role == RoleLegalEmployer || role == RoleAdmin
}
