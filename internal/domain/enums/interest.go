package enums

import "slices"

type Interest string

const (
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
)

func Interests() []Interest {
	return []Interest{InterestProgramming, InterestDesign, InterestMarketing, InterestAccounting, InterestSales, InterestContent, InterestTranslation, InterestTeaching, InterestEngineering, InterestSupport}
}

func (interest Interest) Valid() bool {
	return slices.Contains(Interests(), interest)
}
