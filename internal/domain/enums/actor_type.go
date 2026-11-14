package enums

import "database/sql/driver"

type ActorType uint

const (
	ActorTypeSeeker   ActorType = iota + 1
	ActorTypeParent   ActorType = iota + 1
	ActorTypeEmployer ActorType = iota + 1
	ActorTypeAdmin    ActorType = iota + 1
)

func (value ActorType) String() string {
	switch value {
	case ActorTypeSeeker:
		return "seeker"
	case ActorTypeParent:
		return "parent"
	case ActorTypeEmployer:
		return "employer"
	case ActorTypeAdmin:
		return "admin"
	}
	return ""
}

func GetAllActorTypes() []ActorType {
	return []ActorType{
		ActorTypeSeeker,
		ActorTypeParent,
		ActorTypeEmployer,
		ActorTypeAdmin,
	}
}

func (value ActorType) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *ActorType) Scan(src any) error {
	return scanEnum(src, map[string]ActorType{
		"seeker":   ActorTypeSeeker,
		"parent":   ActorTypeParent,
		"employer": ActorTypeEmployer,
		"admin":    ActorTypeAdmin,
	}, value)
}
