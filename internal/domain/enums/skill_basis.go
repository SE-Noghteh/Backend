package enums

import "database/sql/driver"

type SkillBasis uint

const (
	SkillBasisSelf         SkillBasis = iota + 1
	SkillBasisUsedInCollab SkillBasis = iota + 1
	SkillBasisAssessed     SkillBasis = iota + 1
)

func (value SkillBasis) String() string {
	switch value {
	case SkillBasisSelf:
		return "self"
	case SkillBasisUsedInCollab:
		return "used_in_collab"
	case SkillBasisAssessed:
		return "assessed"
	}
	return ""
}

func GetAllSkillBases() []SkillBasis {
	return []SkillBasis{
		SkillBasisSelf,
		SkillBasisUsedInCollab,
		SkillBasisAssessed,
	}
}

func (value SkillBasis) Value() (driver.Value, error) {
	return value.String(), nil
}

func (value *SkillBasis) Scan(src any) error {
	return scanEnum(src, map[string]SkillBasis{
		"self":           SkillBasisSelf,
		"used_in_collab": SkillBasisUsedInCollab,
		"assessed":       SkillBasisAssessed,
	}, value)
}
