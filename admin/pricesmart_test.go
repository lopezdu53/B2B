package admin

import "testing"

func TestPriceSmartLocations(t *testing.T) {
	t.Parallel()

	list := PriceSmartLocations()
	if len(list) != 3 {
		t.Fatalf("got %d locations, want 3", len(list))
	}

	seen := map[string]bool{}
	for _, p := range list {
		if p.Lat == 0 || p.Lng == 0 || p.Name == "" {
			t.Fatalf("incomplete POI: %+v", p)
		}

		if seen[p.ID] {
			t.Fatalf("duplicate id %s", p.ID)
		}

		seen[p.ID] = true
	}

	if !seen["ps-salitre"] || !seen["ps-usaquen"] || !seen["ps-chia"] {
		t.Fatalf("missing expected clubs: %v", seen)
	}

	byID := map[string]PriceSmartPOI{}
	for _, p := range list {
		byID[p.ID] = p
	}

	salitre := byID["ps-salitre"]
	if salitre.Lat < 4.66 || salitre.Lat > 4.67 || salitre.Lng > -74.10 {
		t.Fatalf("Salitre should sit on Av. Calle 26, got %v,%v", salitre.Lat, salitre.Lng)
	}

	usaquen := byID["ps-usaquen"]
	if usaquen.Lat < 4.74 || usaquen.Lat > 4.76 {
		t.Fatalf("Usaquén should sit on Calle 170, got %v", usaquen.Lat)
	}

	chia := byID["ps-chia"]
	if chia.Lat < 4.88 {
		t.Fatalf("Chía should be in Yerbabuena north of La Caro, got %v", chia.Lat)
	}
}
