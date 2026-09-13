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
}
