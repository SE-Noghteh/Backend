package general

import (
	"zedlancer/backend/internal/application/usecase"
	"zedlancer/backend/internal/domain/enums"
	"zedlancer/backend/internal/domain/valueobjects"
)

type IndividualEmployerSignupRequest struct {
	FirstName        string         `form:"first_name"`
	LastName         string         `form:"last_name"`
	FatherName       string         `form:"father_name"`
	NationalCode     string         `form:"national_code"`
	Gender           enums.Gender   `form:"gender"`
	BirthYear        int            `form:"birth_year"`
	BirthMonth       int            `form:"birth_month"`
	BirthDay         int            `form:"birth_day"`
	Province         enums.Province `form:"province"`
	City             enums.City     `form:"city"`
	Address          string         `form:"address"`
	PostalCode       string         `form:"postal_code"`
	BusinessName     string         `form:"business_name"`
	BusinessActivity string         `form:"business_activity"`
	Website          string         `form:"website"`
	HasLegalGuardian bool           `form:"has_legal_guardian"`
	Phone            string         `form:"phone"`
	Email            string         `form:"email"`
	Password         string         `form:"password"`
}

func (r IndividualEmployerSignupRequest) input() usecase.SignupIndividualEmployer {
	return usecase.SignupIndividualEmployer{
		FirstName: r.FirstName, LastName: r.LastName, FatherName: r.FatherName, NationalCode: r.NationalCode, Gender: r.Gender,
		BirthDate: valueobjects.JalaliDate{Year: r.BirthYear, Month: r.BirthMonth, Day: r.BirthDay}, Province: r.Province, City: r.City, Address: r.Address, PostalCode: r.PostalCode,
		BusinessName: r.BusinessName, BusinessActivity: r.BusinessActivity, Website: r.Website, HasLegalGuardian: r.HasLegalGuardian,
		Phone: r.Phone, Email: r.Email, Password: r.Password,
	}
}
