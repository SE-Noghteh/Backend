package enums

import (
	_ "embed"
	"encoding/json"
	"maps"
	"slices"
)

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

//go:embed iran_locations.json
var iranLocationsJSON []byte

var citiesByProvince, provinceNames, cityNames = loadIranLocations()

func loadIranLocations() (map[Province][]City, []Province, []City) {
	var byProvince map[Province][]City
	if err := json.Unmarshal(iranLocationsJSON, &byProvince); err != nil {
		panic(err)
	}
	uniqueCities := make(map[City]struct{})
	for _, cities := range byProvince {
		slices.Sort(cities)
		for _, city := range cities {
			uniqueCities[city] = struct{}{}
		}
	}
	return byProvince, slices.Sorted(maps.Keys(byProvince)), slices.Sorted(maps.Keys(uniqueCities))
}

func Provinces() []Province { return slices.Clone(provinceNames) }

func (province Province) Valid() bool {
	_, ok := citiesByProvince[province]
	return ok
}

func CitiesInProvince(province Province) []City {
	return slices.Clone(citiesByProvince[province])
}

func CitiesByProvince() map[Province][]City {
	locations := make(map[Province][]City, len(citiesByProvince))
	for province := range citiesByProvince {
		locations[province] = CitiesInProvince(province)
	}
	return locations
}
