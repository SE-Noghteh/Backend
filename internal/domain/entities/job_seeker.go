package entities

import "time"

type City string
type Interest string
type Gender string

const (
	CityTehran          City     = "تهران"
	CityIsfahan         City     = "اصفهان"
	CityMashhad         City     = "مشهد"
	CityShiraz          City     = "شیراز"
	CityTabriz          City     = "تبریز"
	CityKaraj           City     = "کرج"
	CityQom             City     = "قم"
	CityAhvaz           City     = "اهواز"
	CityRasht           City     = "رشت"
	CityYazd            City     = "یزد"
	InterestProgramming Interest = "برنامه نویسی"
	InterestDesign      Interest = "طراحی"
	InterestMarketing   Interest = "بازاریابی"
	InterestAccounting  Interest = "حسابداری"
	InterestSales       Interest = "فروش"
	InterestContent     Interest = "تولید محتوا"
	InterestTranslation Interest = "ترجمه"
	InterestTeaching    Interest = "آموزش"
	InterestEngineering Interest = "مهندسی"
	InterestSupport     Interest = "پشتیبانی"
	GenderFemale        Gender   = "female"
	GenderMale          Gender   = "male"
)

func Cities() []City {
	return []City{CityTehran, CityIsfahan, CityMashhad, CityShiraz, CityTabriz, CityKaraj, CityQom, CityAhvaz, CityRasht, CityYazd}
}
func Interests() []Interest {
	return []Interest{InterestProgramming, InterestDesign, InterestMarketing, InterestAccounting, InterestSales, InterestContent, InterestTranslation, InterestTeaching, InterestEngineering, InterestSupport}
}

type JobSeeker struct {
	ID              uint64     `json:"id"`
	FirstName       string     `json:"first_name"`
	LastName        string     `json:"last_name"`
	NationalCode    string     `json:"national_code"`
	City            City       `json:"city"`
	Address         string     `json:"address"`
	BirthDate       JalaliDate `json:"birth_date"`
	Gender          Gender     `json:"gender"`
	Interests       []Interest `json:"interests"`
	Phone           string     `json:"phone"`
	PasswordHash    string     `json:"-"`
	PhoneVerifiedAt time.Time  `json:"phone_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

type SignupOTP struct {
	Phone        string
	Salt         string
	CodeHash     string
	Attempts     int
	SentAt       time.Time
	ExpiresAt    time.Time
	Registration *JobSeeker
}
