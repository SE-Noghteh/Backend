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
	return []City{CityTehran, CityIsfahan, CityMashhad, CityShiraz, CityTabriz, CityKaraj, CityQom, CityAhvaz, CityRasht, CityYazd}
}

func (city City) Valid() bool {
	return slices.Contains(Cities(), city)
}
