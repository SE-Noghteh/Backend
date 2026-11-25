package enums

import "testing"

func TestIranLocationCatalog(t *testing.T) {
	provinces := Provinces()
	if len(provinces) != 31 {
		t.Fatalf("got %d provinces, want 31", len(provinces))
	}
	total := 0
	for _, province := range provinces {
		cities := CitiesInProvince(province)
		if len(cities) == 0 {
			t.Errorf("province %q has no cities", province)
		}
		total += len(cities)
		for _, city := range cities {
			if !city.Valid() || !city.InProvince(province) {
				t.Errorf("invalid catalog entry: %q, %q", province, city)
			}
		}
	}
	if total != 1458 {
		t.Errorf("got %d province–city choices, want 1458", total)
	}

	const mahabad City = "مهاباد"
	if !mahabad.InProvince("آذربایجان غربی") || !mahabad.InProvince(ProvinceIsfahan) || mahabad.InProvince(ProvinceTehran) {
		t.Error("a shared city name must match only its actual provinces")
	}
}
