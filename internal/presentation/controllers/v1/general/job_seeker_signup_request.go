package general

import (
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type JobSeekerSignupRequest struct {
	FirstName    string           `form:"first_name"`
	LastName     string           `form:"last_name"`
	NationalCode string           `form:"national_code"`
	City         enums.City       `form:"city"`
	Address      string           `form:"address"`
	BirthYear    int              `form:"birth_year"`
	BirthMonth   int              `form:"birth_month"`
	BirthDay     int              `form:"birth_day"`
	Gender       enums.Gender     `form:"gender"`
	Interests    []enums.Interest `form:"interests"`
	Phone        string           `form:"phone"`
	Password     string           `form:"password"`
}

func (r JobSeekerSignupRequest) input() usecase.SignupJobSeeker {
	return usecase.SignupJobSeeker{FirstName: r.FirstName, LastName: r.LastName, NationalCode: r.NationalCode, City: r.City, Address: r.Address,
		BirthDate: valueobjects.JalaliDate{Year: r.BirthYear, Month: r.BirthMonth, Day: r.BirthDay}, Gender: r.Gender, Interests: r.Interests, Phone: r.Phone, Password: r.Password}
}
