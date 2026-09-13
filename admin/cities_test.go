package admin

import "testing"

func TestCanonicalCityCollapsesBogotaVariants(t *testing.T) {
	t.Parallel()

	cases := []string{
		"Bogotá",
		"BOGOTÁ",
		"Bogota",
		"Bogotá, Bogotá",
		"Bogotá, BOGOTÁ",
		"Bogotá, Bogotá D.C.",
		"Bogotá, BOGOTÁ D.C.",
		"Bogotá D.C.",
		"BOGOTÁ D.C.",
		"Bogotá, D.C.",
		"santa fe de bogotá",
	}

	for _, in := range cases {
		if got := CanonicalCity(in); got != DefaultCity {
			t.Errorf("CanonicalCity(%q) = %q, want %q", in, got, DefaultCity)
		}
	}
}

func TestCanonicalCityOtherCities(t *testing.T) {
	t.Parallel()

	if got := CanonicalCity("Medellín"); got != "Medellín" {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalCity("Cali, Valle"); got != "Cali" {
		t.Fatalf("got %q", got)
	}
	if got := CanonicalCity("Boston"); got != "" {
		t.Fatalf("unknown city should be empty, got %q", got)
	}

	if got := CanonicalCity("Chía"); got != "Chía" {
		t.Fatalf("Chía must not collapse into Bogotá, got %q", got)
	}
}

func TestResolveCityFilter(t *testing.T) {
	t.Parallel()

	city, all := ResolveCityFilter("")
	if all || city != DefaultCity {
		t.Fatalf("empty: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("all")
	if !all || city != "" {
		t.Fatalf("all: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("Bogotá, BOGOTÁ D.C.")
	if all || city != DefaultCity {
		t.Fatalf("variant: city=%q all=%v", city, all)
	}
}

func TestBuildSearchWhere(t *testing.T) {
	t.Parallel()

	cases := []struct {
		city, loc, bar, want string
	}{
		{"", "", "", "Bogotá"},
		{"Bogotá", "", "", "Bogotá"},
		{"Bogotá", "Usaquén", "", "Usaquén, Bogotá"},
		{"Bogotá", "Usaquén", "El Redil", "El Redil, Usaquén, Bogotá"},
		{"bogota dc", "Chapinero", "", "Chapinero, Bogotá"},
		{"Medellín", "", "", "Medellín"},
		{"Medellín", "El Poblado", "", "El Poblado, Medellín"},
	}

	for _, tc := range cases {
		if got := BuildSearchWhere(tc.city, tc.loc, tc.bar); got != tc.want {
			t.Errorf("BuildSearchWhere(%q,%q,%q)=%q want %q", tc.city, tc.loc, tc.bar, got, tc.want)
		}
	}
}

func TestSearchKeyword(t *testing.T) {
	t.Parallel()

	got := SearchKeyword("restaurantes", "Bogotá", "Usaquén", "", "")
	if got != "restaurantes en Usaquén, Bogotá" {
		t.Fatalf("structured: %q", got)
	}

	got = SearchKeyword("hoteles", "", "", "", "Laureles, Medellín")
	if got != "hoteles en Laureles, Medellín" {
		t.Fatalf("legacy where: %q", got)
	}

	got = SearchKeyword("cafés", "", "", "", "")
	if got != "cafés en Bogotá" {
		t.Fatalf("default city: %q", got)
	}

	got = SearchKeyword("farmacias", "Bogotá", "Suba", "Lisboa", "ignored")
	if got != "farmacias en Lisboa, Suba, Bogotá" {
		t.Fatalf("barrio wins over where: %q", got)
	}
}

func TestCanonicalCityLocalidadIsBogota(t *testing.T) {
	t.Parallel()

	for _, in := range []string{"Usaquén", "Chapinero", "Barrios Unidos", "Kennedy", "La Candelaria"} {
		if got := CanonicalCity(in); got != DefaultCity {
			t.Errorf("CanonicalCity(%q) = %q, want %q", in, got, DefaultCity)
		}
	}

	if got := CanonicalCity("Chía"); got != "Chía" {
		t.Fatalf("Chía must stay its own city, got %q", got)
	}

	if got := CanonicalCity("Soacha"); got != "Soacha" {
		t.Fatalf("Soacha must stay its own city, got %q", got)
	}
}

func TestMatchesCityFilterBogotaLocalidad(t *testing.T) {
	t.Parallel()

	if !MatchesCityFilter("Bogotá", "Usaquén", "", "", "Cra 7 #127, Bogotá") {
		t.Fatal("Usaquén should match Bogotá")
	}

	if !MatchesCityFilter("Bogotá", "Barrios Unidos", "Bogotá", "", "") {
		t.Fatal("Barrios Unidos should match Bogotá")
	}

	if !MatchesCityFilter("Bogotá", "Bogotá, BOGOTÁ D.C.", "", "", "") {
		t.Fatal("canonical Bogotá should match")
	}

	if !MatchesCityFilter("Bogotá", "", "Bogotá D.C.", "", "Calle 72") {
		t.Fatal("empty city + Bogotá state should match")
	}

	if MatchesCityFilter("Bogotá", "Chía", "Cundinamarca", "", "cerca de Bogotá") {
		t.Fatal("Chía must not match Bogotá")
	}

	if MatchesCityFilter("Medellín", "Usaquén", "", "", "") {
		t.Fatal("Usaquén must not match Medellín")
	}

	if !MatchesCityFilter("Cali", "Cali, Valle", "", "", "") {
		t.Fatal("Cali, Valle should match Cali")
	}

	if !MatchesCityFilter("", "Boston", "", "", "") {
		t.Fatal("empty filter matches everything")
	}
}

func TestIsBogotaPlace(t *testing.T) {
	t.Parallel()

	if !IsBogotaPlace("Usaquén", "", "", "") {
		t.Fatal("Usaquén is Bogotá")
	}

	if IsBogotaPlace("Chía", "", "", "Autopista Norte") {
		t.Fatal("Chía is not Bogotá")
	}

	if !IsBogotaPlace("", "Bogotá", "", "Calle 100") {
		t.Fatal("state Bogotá should count")
	}
}

func TestCityListStartsWithBogota(t *testing.T) {
	t.Parallel()

	list := CityList()
	if len(list) < 2 || list[0] != DefaultCity {
		t.Fatalf("first city = %v", list)
	}

	seen := map[string]struct{}{}
	for _, c := range list {
		if _, ok := seen[c]; ok {
			t.Fatalf("duplicate city %q", c)
		}
		seen[c] = struct{}{}
	}
}
