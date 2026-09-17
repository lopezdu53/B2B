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
	if !all || city != "" {
		t.Fatalf("empty: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("all")
	if !all || city != "" {
		t.Fatalf("all: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("todos")
	if !all || city != "" {
		t.Fatalf("todos: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("Bogotá, BOGOTÁ D.C.")
	if all || city != DefaultCity {
		t.Fatalf("variant: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("Cundinamarca")
	if all || city != CundinamarcaRegion {
		t.Fatalf("dept as city: city=%q all=%v", city, all)
	}

	city, all = ResolveCityFilter("Boyacá")
	if all || city != BoyacaRegion {
		t.Fatalf("boyaca: city=%q all=%v", city, all)
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

func TestClaudeSearchKeyword(t *testing.T) {
	t.Parallel()

	got := ClaudeSearchKeyword("restaurantes", "Usaquén, Bogotá")
	if got != "restaurantes en Usaquén, Bogotá" {
		t.Fatalf("got %q", got)
	}

	got = ClaudeSearchKeyword("hoteles", "")
	if got != "hoteles" {
		t.Fatalf("no where: %q", got)
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

	if got := CanonicalCity("La Calera"); got != "La Calera" {
		t.Fatalf("La Calera must stay its own city, got %q", got)
	}

	if got := CanonicalCity("calera"); got != "La Calera" {
		t.Fatalf("calera alias, got %q", got)
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

	if MatchesCityFilter("Bogotá", "La Calera", "Cundinamarca", "", "La Calera") {
		t.Fatal("La Calera must not match Bogotá")
	}

	if !MatchesCityFilter("Cundinamarca", "Bogotá", "", "", "") {
		t.Fatal("Bogotá should match Cundinamarca")
	}

	if !MatchesCityFilter("Cundinamarca", "Chía", "Cundinamarca", "", "") {
		t.Fatal("Chía should match Cundinamarca")
	}

	if !MatchesCityFilter("Cundinamarca", "La Calera", "Cundinamarca", "", "") {
		t.Fatal("La Calera should match Cundinamarca")
	}

	if !MatchesCityFilter("Cundinamarca", "Usaquén", "", "", "Cra 7") {
		t.Fatal("Usaquén should match Cundinamarca")
	}

	if MatchesCityFilter("Cundinamarca", "Tunja", "Boyacá", "", "") {
		t.Fatal("Tunja must not match Cundinamarca")
	}

	if !MatchesCityFilter("Boyacá", "Tunja", "", "", "") {
		t.Fatal("Tunja should match Boyacá")
	}

	if !MatchesCityFilter("Boyacá", "Duitama", "Boyacá", "", "") {
		t.Fatal("Duitama should match Boyacá")
	}

	if !MatchesCityFilter("Boyacá", "Villa de Leyva", "", "", "") {
		t.Fatal("Villa de Leyva should match Boyacá")
	}

	if MatchesCityFilter("Boyacá", "Bogotá", "", "", "") {
		t.Fatal("Bogotá must not match Boyacá")
	}

	if !MatchesCityFilter("Tunja", "Tunja", "Boyacá", "", "") {
		t.Fatal("Tunja should match Tunja")
	}

	if !MatchesCityFilter("Chía", "Chía", "Cundinamarca", "", "") {
		t.Fatal("Chía should match Chía")
	}

	if !MatchesCityFilter("Chía", "Cundinamarca", "", "Chía", "Calle 12") {
		t.Fatal("Chía borough should match Chía filter")
	}

	if !MatchesCityFilter("La Calera", "La Calera", "", "", "") {
		t.Fatal("La Calera should match itself")
	}

	if !MatchesCityFilter("La Calera", "Cundinamarca", "", "", "Vereda El Hato, La Calera") {
		t.Fatal("La Calera address should match La Calera filter")
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

	if IsBogotaPlace("La Calera", "Cundinamarca", "", "") {
		t.Fatal("La Calera is not Bogotá")
	}

	if IsBogotaPlace("", "", "", "Centro, La Calera cerca de Bogotá") {
		t.Fatal("La Calera address must not count as Bogotá")
	}

	if !IsBogotaPlace("", "Bogotá", "", "Calle 100") {
		t.Fatal("state Bogotá should count")
	}
}

func TestCityListIsCundinamarcaAndBoyaca(t *testing.T) {
	t.Parallel()

	list := CityList()
	if len(list) < 3 || list[0] != DefaultCity {
		t.Fatalf("first cities = %v", list)
	}

	seen := map[string]struct{}{}
	for _, c := range list {
		if _, ok := seen[c]; ok {
			t.Fatalf("duplicate city %q", c)
		}
		seen[c] = struct{}{}
	}

	for _, need := range []string{"Bogotá", "Chía", "La Calera", "Cajicá", "Cota", "Tunja", "Duitama", "Sogamoso", "Villa de Leyva"} {
		if _, ok := seen[need]; !ok {
			t.Fatalf("city list missing %q", need)
		}
	}

	for _, gone := range []string{"Cundinamarca", "Boyacá", "Medellín", "Cali", "Barranquilla"} {
		if _, ok := seen[gone]; ok {
			t.Fatalf("city list should not include %q", gone)
		}
	}

	depts := Departments()
	if len(depts) != 2 || depts[0].Name != CundinamarcaRegion || depts[1].Name != BoyacaRegion {
		t.Fatalf("departments: %+v", depts)
	}

	if got := CitiesForDepartment(CundinamarcaRegion); got[0] != DefaultCity {
		t.Fatalf("cundinamarca cities: %v", got)
	}

	if got := CitiesForDepartment(BoyacaRegion); got[0] != "Tunja" {
		t.Fatalf("boyaca cities: %v", got)
	}
}

func TestInferCityFromSearch(t *testing.T) {
	t.Parallel()

	cases := []struct{ what, where, want string }{
		{"restaurantes", "chia", "Cundinamarca"},
		{"restaurantes", "Chía", "Cundinamarca"},
		{"restaurantes en chia", "", "Cundinamarca"},
		{"restaurantes", "La Calera", "Cundinamarca"},
		{"hoteles en la calera", "", "Cundinamarca"},
		{"restaurantes", "Usaquén, Bogotá", "Cundinamarca"},
		{"restaurantes", "calera", "Cundinamarca"},
		{"restaurantes, hoteles", "Cajicá", "Cundinamarca"},
		{"hoteles", "Medellín", "Medellín"},
		{"hoteles", "Tunja", "Boyacá"},
		{"restaurantes", "Duitama", "Boyacá"},
	}

	for _, tc := range cases {
		if got := InferCityFromSearch(tc.what, tc.where); got != tc.want {
			t.Errorf("InferCityFromSearch(%q, %q)=%q want %q", tc.what, tc.where, got, tc.want)
		}
	}
}

func TestCityFilterFromInputsRemembersSearch(t *testing.T) {
	t.Parallel()

	city, all, val := CityFilterFromInputs("", "Chía")
	if all || city != "Chía" || val != "Chía" {
		t.Fatalf("remembered Chía: city=%q all=%v val=%q", city, all, val)
	}

	city, all, val = CityFilterFromInputs("", "")
	if !all || city != "" || val != PlaceAll {
		t.Fatalf("default todos: city=%q all=%v val=%q", city, all, val)
	}

	city, all, val = CityFilterFromInputs("La Calera", "Chía")
	if all || city != "La Calera" || val != "La Calera" {
		t.Fatalf("query wins: city=%q all=%v val=%q", city, all, val)
	}

	city, all, val = CityFilterFromInputs("all", "Chía")
	if !all || city != "" || val != PlaceAll {
		t.Fatalf("all: city=%q all=%v val=%q", city, all, val)
	}

	filter, dept, cityVal, all := PlaceFilterFromInputs("", "", "", "")
	if !all || filter != "" || dept != PlaceAll || cityVal != PlaceAll {
		t.Fatalf("empty place: filter=%q dept=%q city=%q all=%v", filter, dept, cityVal, all)
	}

	filter, dept, cityVal, all = PlaceFilterFromInputs("Cundinamarca", "all", "", "")
	if all || filter != CundinamarcaRegion || dept != CundinamarcaRegion || cityVal != PlaceAll {
		t.Fatalf("cund todos: filter=%q dept=%q city=%q all=%v", filter, dept, cityVal, all)
	}

	filter, dept, cityVal, all = PlaceFilterFromInputs("Boyacá", "all", "", "")
	if all || filter != BoyacaRegion || dept != BoyacaRegion || cityVal != PlaceAll {
		t.Fatalf("boyaca todos: filter=%q dept=%q city=%q all=%v", filter, dept, cityVal, all)
	}

	filter, dept, cityVal, all = PlaceFilterFromInputs("Boyacá", "Tunja", "", "")
	if all || filter != "Tunja" || dept != BoyacaRegion || cityVal != "Tunja" {
		t.Fatalf("tunja: filter=%q dept=%q city=%q all=%v", filter, dept, cityVal, all)
	}

	filter, dept, cityVal, all = PlaceFilterFromInputs("", "Cundinamarca", "", "")
	if all || filter != CundinamarcaRegion || dept != CundinamarcaRegion || cityVal != PlaceAll {
		t.Fatalf("legacy cund cookie: filter=%q dept=%q city=%q all=%v", filter, dept, cityVal, all)
	}

	zBog := Zone{Name: "Norte", City: "Bogotá"}
	zTun := Zone{Name: "Centro", City: "Tunja"}
	if !ZoneMatchesPlaceFilter(zBog, PlaceAll, PlaceAll) || !ZoneMatchesPlaceFilter(zTun, PlaceAll, PlaceAll) {
		t.Fatal("todos should show every zone")
	}

	if !ZoneMatchesPlaceFilter(zBog, CundinamarcaRegion, PlaceAll) || ZoneMatchesPlaceFilter(zTun, CundinamarcaRegion, PlaceAll) {
		t.Fatal("cundinamarca should hide Boyacá zones")
	}

	if ZoneMatchesPlaceFilter(zBog, BoyacaRegion, PlaceAll) || !ZoneMatchesPlaceFilter(zTun, BoyacaRegion, PlaceAll) {
		t.Fatal("boyaca should hide Cundinamarca zones")
	}

	if !ZoneMatchesPlaceFilter(zTun, BoyacaRegion, "Tunja") || ZoneMatchesPlaceFilter(zBog, BoyacaRegion, "Tunja") {
		t.Fatal("tunja city should only keep Tunja zones")
	}
}
