package enums

import "slices"

type Gender string

const (
	GenderFemale Gender = "female"
	GenderMale   Gender = "male"
)

func Genders() []Gender {
	return []Gender{GenderFemale, GenderMale}
}

func (gender Gender) Valid() bool {
	return slices.Contains(Genders(), gender)
}
