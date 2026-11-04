package enums

import "slices"

type Province string

const (
	ProvinceTehran         Province = "تهران"
	ProvinceIsfahan        Province = "اصفهان"
	ProvinceKhorasanRazavi Province = "خراسان رضوی"
	ProvinceFars           Province = "فارس"
	ProvinceEastAzerbaijan Province = "آذربایجان شرقی"
	ProvinceAlborz         Province = "البرز"
	ProvinceQom            Province = "قم"
	ProvinceKhuzestan      Province = "خوزستان"
	ProvinceGilan          Province = "گیلان"
	ProvinceYazd           Province = "یزد"
)

func Provinces() []Province {
	return []Province{ProvinceTehran, ProvinceIsfahan, ProvinceKhorasanRazavi, ProvinceFars, ProvinceEastAzerbaijan, ProvinceAlborz, ProvinceQom, ProvinceKhuzestan, ProvinceGilan, ProvinceYazd}
}

func (province Province) Valid() bool { return slices.Contains(Provinces(), province) }

func (city City) Province() Province {
	switch city {
	case CityTehran:
		return ProvinceTehran
	case CityIsfahan:
		return ProvinceIsfahan
	case CityMashhad:
		return ProvinceKhorasanRazavi
	case CityShiraz:
		return ProvinceFars
	case CityTabriz:
		return ProvinceEastAzerbaijan
	case CityKaraj:
		return ProvinceAlborz
	case CityQom:
		return ProvinceQom
	case CityAhvaz:
		return ProvinceKhuzestan
	case CityRasht:
		return ProvinceGilan
	case CityYazd:
		return ProvinceYazd
	default:
		return ""
	}
}
