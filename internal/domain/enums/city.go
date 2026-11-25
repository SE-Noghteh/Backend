package enums

import "slices"

type City string

const (
	CityTehran  City = "تهران"
	CityIsfahan City = "اصفهان"
	CityMashhad City = "مشهد"
	CityShiraz  City = "شیراز"
	CityTabriz  City = "تبریز"
	CityKaraj   City = "کرج"
	CityQom     City = "قم"
	CityAhvaz   City = "اهواز"
	CityRasht   City = "رشت"
	CityYazd    City = "یزد"
)

func Cities() []City {
	return slices.Clone(cityNames)
}

func (city City) Valid() bool {
	_, ok := slices.BinarySearch(cityNames, city)
	return ok
}

func (city City) InProvince(province Province) bool {
	_, ok := slices.BinarySearch(citiesByProvince[province], city)
	return ok
}
