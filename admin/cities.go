package admin

import (
	"strings"
	"unicode"
)

// DefaultCity is the urban capital used for Bogotá localidad matching.
const DefaultCity = "Bogotá"

// CundinamarcaRegion is the Cundinamarca department filter: Bogotá D.C. plus
// surrounding municipalities (Chía, La Calera, Cajicá, Soacha, …).
const CundinamarcaRegion = "Cundinamarca"

// BoyacaRegion is the Boyacá department filter (Tunja, Duitama, Sogamoso, …).
const BoyacaRegion = "Boyacá"

// PlaceAll is the dropdown value that shows every business and zone.
const PlaceAll = "all"

// CundinamarcaCities are the municipalities offered under Cundinamarca.
var CundinamarcaCities = []string{
	"Bogotá",
	"Soacha",
	"Chía",
	"Zipaquirá",
	"Facatativá",
	"Fusagasugá",
	"Girardot",
	"Cajicá",
	"Cota",
	"La Calera",
	"Mosquera",
	"Funza",
	"Madrid",
	"Sopó",
	"Tabio",
	"Tenjo",
	"Sibaté",
	"Tocancipá",
	"Gachancipá",
	"Cogua",
}

// BoyacaCities are the municipalities offered under Boyacá.
var BoyacaCities = []string{
	"Tunja",
	"Duitama",
	"Sogamoso",
	"Chiquinquirá",
	"Paipa",
	"Villa de Leyva",
	"Puerto Boyacá",
	"Nobsa",
	"Moniquirá",
	"Tibasosa",
	"Ráquira",
	"Samacá",
	"Combita",
	"Garagoa",
	"Soatá",
	"Santa Rosa de Viterbo",
	"Monguí",
	"Toca",
	"Belén",
	"Guateque",
}

// PredefinedCities is every city in the two supported departments.
var PredefinedCities = append(append([]string{}, CundinamarcaCities...), BoyacaCities...)

// PlaceDepartment is a department and the cities shown under it.
type PlaceDepartment struct {
	Name   string   `json:"name"`
	Cities []string `json:"cities"`
}

// Departments is the map filter: Cundinamarca and Boyacá only.
func Departments() []PlaceDepartment {
	return []PlaceDepartment{
		{Name: CundinamarcaRegion, Cities: append([]string{}, CundinamarcaCities...)},
		{Name: BoyacaRegion, Cities: append([]string{}, BoyacaCities...)},
	}
}

// DepartmentNames returns the two supported departments.
func DepartmentNames() []string {
	return []string{CundinamarcaRegion, BoyacaRegion}
}

var cityByKey map[string]string

func init() {
	cityByKey = make(map[string]string, len(PredefinedCities)+32)
	for _, name := range PredefinedCities {
		cityByKey[cityKey(name)] = name
	}

	for _, name := range []string{
		"Medellín", "Cali", "Barranquilla", "Cartagena", "Bucaramanga",
		"Pereira", "Cúcuta", "Santa Marta", "Ibagué", "Manizales",
		"Villavicencio", "Pasto", "Neiva", "Armenia", "Montería",
		"Valledupar", "Sincelejo", "Popayán", "Riohacha", "Quibdó",
		"Florencia", "Yopal",
	} {
		if k := cityKey(name); k != "" {
			if _, exists := cityByKey[k]; !exists {
				cityByKey[k] = name
			}
		}
	}

	// Common scrape variants that should collapse into a predefined city.
	for _, alias := range []string{
		"bogota dc", "bogota d c", "bogota d.c", "bogota, bogota",
		"santa fe de bogota", "santafede bogota", "bogota colombia",
		"bogota d.c.", "bogota, d.c.", "bogota, dc",
	} {
		cityByKey[cityKey(alias)] = DefaultCity
	}

	cityByKey[cityKey("calera")] = "La Calera"
	cityByKey[cityKey("la calera cundinamarca")] = "La Calera"
	cityByKey[cityKey("villa de leiva")] = "Villa de Leyva"
	cityByKey[cityKey("villa de leyva boyaca")] = "Villa de Leyva"
	cityByKey[cityKey(CundinamarcaRegion)] = CundinamarcaRegion
	cityByKey[cityKey("cundinamarca colombia")] = CundinamarcaRegion
	cityByKey[cityKey(BoyacaRegion)] = BoyacaRegion
	cityByKey[cityKey("boyaca colombia")] = BoyacaRegion
	cityByKey[cityKey("boyaca")] = BoyacaRegion

	// Google often stores the locality (Usaquén, Chapinero…) as "city".
	// Those still belong to Bogotá for the map/list filter.
	for _, loc := range BogotaUrbanLocalidades() {
		if k := cityKey(loc.Nombre); k != "" {
			if _, exists := cityByKey[k]; !exists {
				cityByKey[k] = DefaultCity
			}
		}
	}
}

// CityList returns cities of Cundinamarca and Boyacá (no department names).
func CityList() []string {
	out := make([]string, 0, len(CundinamarcaCities)+len(BoyacaCities))
	seen := map[string]struct{}{}

	for _, name := range append(append([]string{}, CundinamarcaCities...), BoyacaCities...) {
		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}
		out = append(out, name)
	}

	return out
}

// CitiesForDepartment returns the city dropdown for one department.
func CitiesForDepartment(department string) []string {
	switch NormalizeDepartment(department) {
	case CundinamarcaRegion:
		return append([]string{}, CundinamarcaCities...)
	case BoyacaRegion:
		return append([]string{}, BoyacaCities...)
	default:
		return CityList()
	}
}

// CanonicalCity maps a scraped city string to a single display name.
// "Bogotá, BOGOTÁ D.C.", "BOGOTÁ" and "Bogotá D.C." all become "Bogotá".
func CanonicalCity(s string) string {
	key := cityKey(s)
	if key == "" {
		return ""
	}

	if name, ok := cityByKey[key]; ok {
		return name
	}

	if tok := firstToken(key); tok != "" {
		if name, ok := cityByKey[tok]; ok {
			return name
		}
	}

	return ""
}

// BuildSearchWhere composes the Google Maps "en …" location from the
// cascaded search dropdowns: barrio, localidad, city (most specific first).
// Empty city defaults to Bogotá. Unknown city names are kept as typed.
func BuildSearchWhere(city, localidad, barrio string) string {
	city = strings.TrimSpace(city)
	if canon := CanonicalCity(city); canon != "" {
		city = canon
	}

	if city == "" {
		city = DefaultCity
	}

	localidad = strings.TrimSpace(localidad)
	barrio = strings.TrimSpace(barrio)

	parts := make([]string, 0, 3)
	if barrio != "" {
		parts = append(parts, barrio)
	}

	if localidad != "" {
		parts = append(parts, localidad)
	}

	parts = append(parts, city)

	return strings.Join(parts, ", ")
}

// SearchKeyword builds the scrape job keyword from the form fields.
// Structured city/localidad/barrio win over the legacy free-text "where".
func SearchKeyword(what, city, localidad, barrio, where string) string {
	what = strings.TrimSpace(what)
	loc := ""

	if strings.TrimSpace(city) != "" || strings.TrimSpace(localidad) != "" || strings.TrimSpace(barrio) != "" {
		loc = BuildSearchWhere(city, localidad, barrio)
	} else {
		loc = strings.TrimSpace(where)
	}

	if loc == "" {
		loc = DefaultCity
	}

	if what == "" {
		return loc
	}

	return what + " en " + loc
}

// isChoiceAll reports whether a dropdown value means "show everything".
func isChoiceAll(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", PlaceAll, "todos", "todas", "*":
		return true
	default:
		return false
	}
}

// NormalizeDepartment maps a dropdown/query value to Cundinamarca, Boyacá, or all.
func NormalizeDepartment(raw string) string {
	if isChoiceAll(raw) {
		return PlaceAll
	}

	switch cityKey(raw) {
	case "cundinamarca":
		return CundinamarcaRegion
	case "boyaca":
		return BoyacaRegion
	default:
		return PlaceAll
	}
}

// NormalizeCityChoice maps a city dropdown value to a canonical city or all.
func NormalizeCityChoice(raw string) string {
	if isChoiceAll(raw) {
		return PlaceAll
	}

	if c := CanonicalCity(raw); c != "" && c != CundinamarcaRegion && c != BoyacaRegion {
		return c
	}

	if c := CanonicalCity(raw); c == CundinamarcaRegion || c == BoyacaRegion {
		return PlaceAll
	}

	return strings.TrimSpace(raw)
}

// ResolveCityFilter interprets a city query parameter.
// "all" / "todos" / "todas" / "*" means no city filter. Empty also means all.
func ResolveCityFilter(raw string) (city string, all bool) {
	if isChoiceAll(raw) {
		return "", true
	}

	if isCundinamarcaFilter(raw) {
		return CundinamarcaRegion, false
	}

	if isBoyacaFilter(raw) {
		return BoyacaRegion, false
	}

	if c := CanonicalCity(raw); c != "" {
		return c, false
	}

	return strings.TrimSpace(raw), false
}

// ResolvePlaceFilter combines department + city dropdowns into the store filter.
// City wins when it is a specific municipality. Department applies when city is Todos.
func ResolvePlaceFilter(department, city string) (filter string, all bool) {
	cityChoice := NormalizeCityChoice(city)
	if cityChoice != PlaceAll {
		return ResolveCityFilter(cityChoice)
	}

	switch NormalizeDepartment(department) {
	case CundinamarcaRegion:
		return CundinamarcaRegion, false
	case BoyacaRegion:
		return BoyacaRegion, false
	default:
		return "", true
	}
}

func cityKey(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	prevSpace := true

	for _, r := range strings.ToLower(s) {
		switch r {
		case 'á', 'à':
			r = 'a'
		case 'é', 'è':
			r = 'e'
		case 'í', 'ì':
			r = 'i'
		case 'ó', 'ò':
			r = 'o'
		case 'ú', 'ù', 'ü':
			r = 'u'
		case 'ñ':
			r = 'n'
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevSpace = false
			continue
		}

		if !prevSpace {
			b.WriteByte(' ')
			prevSpace = true
		}
	}

	return strings.TrimSpace(b.String())
}

func firstToken(key string) string {
	if i := strings.IndexByte(key, ' '); i > 0 {
		return key[:i]
	}

	return key
}

// BogotaCityAliasKeys are folded names that count as Bogotá (the city itself
// plus urban localidades Google may emit in complete_address.city).
func BogotaCityAliasKeys() []string {
	out := []string{"bogota", "bogota dc", "bogota d c", "santa fe de bogota"}
	seen := map[string]struct{}{
		"bogota": {}, "bogota dc": {}, "bogota d c": {}, "santa fe de bogota": {},
	}

	for _, loc := range BogotaUrbanLocalidades() {
		k := cityKey(loc.Nombre)
		if k == "" {
			continue
		}

		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, k)
	}

	return out
}

// BogotaSatelliteTownKeys are nearby municipalities that must not match a
// Bogotá city filter even if the address mentions the capital.
func BogotaSatelliteTownKeys() []string {
	return []string{
		"chia", "soacha", "zipaquira", "facatativa", "cajica",
		"cota", "mosquera", "funza", "madrid", "girardot",
		"la calera", "calera", "fusagasuga", "sopo", "tabio",
		"tenjo", "sibate", "tocancipa", "gachancipa", "cogua",
	}
}

func isSatelliteTown(s string) bool {
	k := cityKey(s)
	if k == "" {
		return false
	}

	padded := " " + k + " "
	for _, sat := range BogotaSatelliteTownKeys() {
		if k == sat || firstToken(k) == sat || strings.HasPrefix(k, sat+" ") || strings.Contains(padded, " "+sat+" ") {
			return true
		}
	}

	return false
}

// IsBogotaPlace reports whether a scraped address belongs to Bogotá.
// Google frequently puts the localidad in the city field ("Usaquén").
func IsBogotaPlace(city, state, borough, address string) bool {
	if isSatelliteTown(city) || isSatelliteTown(borough) || isSatelliteTown(address) {
		return false
	}

	if CanonicalCity(city) == DefaultCity {
		return true
	}

	if CanonicalCity(state) == DefaultCity {
		return true
	}

	if CanonicalCity(borough) == DefaultCity {
		return true
	}

	if cityKey(city) == "" && strings.Contains(cityKey(address), "bogota") {
		return true
	}

	return false
}

// InCundinamarcaRegion reports whether a city name belongs to the default
// Cundinamarca map (Bogotá D.C. and neighboring municipalities).
func InCundinamarcaRegion(name string) bool {
	k := cityKey(name)
	if k == "" {
		return false
	}

	if k == "cundinamarca" || strings.Contains(k, "cundinamarca") {
		return true
	}

	canon := CanonicalCity(name)
	if canon == DefaultCity || canon == CundinamarcaRegion {
		return true
	}

	return isSatelliteTown(name)
}

// BoyacaTownKeys are folded municipality names that count as Boyacá.
func BoyacaTownKeys() []string {
	out := make([]string, 0, len(BoyacaCities)+2)
	seen := map[string]struct{}{}

	for _, name := range BoyacaCities {
		k := cityKey(name)
		if k == "" {
			continue
		}

		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, k)
	}

	for _, alias := range []string{"villa de leiva", "villa de leyva"} {
		if _, ok := seen[alias]; ok {
			continue
		}

		seen[alias] = struct{}{}
		out = append(out, alias)
	}

	return out
}

func isBoyacaTown(s string) bool {
	k := cityKey(s)
	if k == "" {
		return false
	}

	padded := " " + k + " "
	for _, town := range BoyacaTownKeys() {
		if k == town || firstToken(k) == town || strings.HasPrefix(k, town+" ") || strings.Contains(padded, " "+town+" ") {
			return true
		}
	}

	return false
}

// InBoyacaRegion reports whether a place name belongs to Boyacá.
func InBoyacaRegion(name string) bool {
	k := cityKey(name)
	if k == "" {
		return false
	}

	if k == "boyaca" || strings.Contains(k, "boyaca") {
		return true
	}

	canon := CanonicalCity(name)
	if canon == BoyacaRegion {
		return true
	}

	return isBoyacaTown(name)
}

func isBoyacaFilter(filter string) bool {
	k := cityKey(filter)
	if canon := CanonicalCity(filter); canon != "" {
		k = cityKey(canon)
	}

	return k == "boyaca"
}

// IsBoyacaPlace reports whether a scraped address is in Boyacá.
func IsBoyacaPlace(city, state, borough, address string) bool {
	if isBoyacaTown(city) || isBoyacaTown(borough) || isBoyacaTown(address) {
		return true
	}

	if InBoyacaRegion(city) || InBoyacaRegion(borough) || InBoyacaRegion(state) {
		return true
	}

	blob := cityKey(strings.Join([]string{city, state, borough, address}, " "))

	return strings.Contains(blob, "boyaca")
}

// DepartmentOfCity returns Cundinamarca or Boyacá for a city name, or empty.
func DepartmentOfCity(city string) string {
	if isCundinamarcaFilter(city) || InCundinamarcaRegion(city) {
		return CundinamarcaRegion
	}

	if isBoyacaFilter(city) || InBoyacaRegion(city) {
		return BoyacaRegion
	}

	return ""
}

// ZoneMatchesPlaceFilter reports whether a drawn zone belongs to the selected
// department/city (Todos matches every zone).
func ZoneMatchesPlaceFilter(z Zone, department, city string) bool {
	filter, all := ResolvePlaceFilter(department, city)
	if all {
		return true
	}

	if isCundinamarcaFilter(filter) {
		return InCundinamarcaRegion(z.City) || cityKey(z.City) == "cundinamarca"
	}

	if isBoyacaFilter(filter) {
		return InBoyacaRegion(z.City) || cityKey(z.City) == "boyaca"
	}

	return MatchesCityFilter(filter, z.City, "", "", z.City)
}

func isCundinamarcaFilter(filter string) bool {
	k := cityKey(filter)
	if canon := CanonicalCity(filter); canon != "" {
		k = cityKey(canon)
	}

	return k == "cundinamarca"
}

// IsCundinamarcaPlace reports whether a scraped address is in Bogotá D.C. or
// Cundinamarca (Chía, La Calera, Cajicá, Soacha, …).
func IsCundinamarcaPlace(city, state, borough, address string) bool {
	if IsBogotaPlace(city, state, borough, address) {
		return true
	}

	if isSatelliteTown(city) || isSatelliteTown(borough) || isSatelliteTown(address) {
		return true
	}

	if InCundinamarcaRegion(city) || InCundinamarcaRegion(borough) || InCundinamarcaRegion(state) {
		return true
	}

	blob := cityKey(strings.Join([]string{city, state, borough, address}, " "))

	return strings.Contains(blob, "cundinamarca")
}

// MatchesCityFilter is the Go equivalent of the map/list city SQL.
// An empty filter matches everything. "Cundinamarca" is Bogotá plus nearby
// towns. "Bogotá" stays urban-only (no Chía / La Calera).
func MatchesCityFilter(filter, city, state, borough, address string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}

	if isCundinamarcaFilter(filter) {
		return IsCundinamarcaPlace(city, state, borough, address)
	}

	if isBoyacaFilter(filter) {
		return IsBoyacaPlace(city, state, borough, address)
	}

	if cityFieldMatchesFilter(filter, city) {
		return true
	}

	wantBogota := firstToken(cityKey(filter)) == "bogota" || CanonicalCity(filter) == DefaultCity
	if wantBogota {
		return IsBogotaPlace(city, state, borough, address)
	}

	if cityFieldMatchesFilter(filter, borough) {
		return true
	}

	fk := cityKey(filter)
	if canon := CanonicalCity(filter); canon != "" {
		fk = cityKey(canon)
	}

	return fk != "" && strings.Contains(cityKey(address), fk)
}

func cityFieldMatchesFilter(filter, scraped string) bool {
	fk := cityKey(filter)
	if canon := CanonicalCity(filter); canon != "" {
		fk = cityKey(canon)
	}

	ck := cityKey(scraped)
	if fk == "" || ck == "" {
		return false
	}

	if ck == fk || strings.HasPrefix(ck, fk+" ") {
		return true
	}

	if canon := CanonicalCity(scraped); canon != "" && cityKey(canon) == fk {
		return true
	}

	return false
}

// InferCityFromSearch picks the city dropdown for a dashboard search.
// Searches in a Cundinamarca or Boyacá town keep the department view.
func InferCityFromSearch(what, where string) string {
	dept, city := InferPlaceFromSearch(what, where)
	if city != "" && city != PlaceAll {
		return city
	}

	if dept != "" && dept != PlaceAll {
		return dept
	}

	return ""
}

// InferPlaceFromSearch returns department + city for the map filters after a
// search. Towns in a supported department widen to that department + Todos.
func InferPlaceFromSearch(what, where string) (department, city string) {
	c := inferCityFromText(where)
	if c == "" {
		c = inferCityFromText(what)
	}

	if c == "" {
		return "", ""
	}

	if isCundinamarcaFilter(c) || InCundinamarcaRegion(c) {
		return CundinamarcaRegion, PlaceAll
	}

	if isBoyacaFilter(c) || InBoyacaRegion(c) {
		return BoyacaRegion, PlaceAll
	}

	if d := DepartmentOfCity(c); d != "" {
		return d, c
	}

	return PlaceAll, c
}

func inferCityFromText(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}

	lower := strings.ToLower(s)
	if i := strings.LastIndex(lower, " en "); i >= 0 {
		if c := inferCityFromText(s[i+4:]); c != "" {
			return c
		}
	}

	if c := CanonicalCity(s); c != "" {
		return c
	}

	parts := strings.Split(s, ",")
	for i := len(parts) - 1; i >= 0; i-- {
		if c := CanonicalCity(strings.TrimSpace(parts[i])); c != "" {
			return c
		}
	}

	tokens := strings.Fields(cityKey(s))
	for n := len(tokens); n >= 1; n-- {
		if name, ok := cityByKey[strings.Join(tokens[:n], " ")]; ok {
			return name
		}
	}

	for n := 0; n < len(tokens); n++ {
		if name, ok := cityByKey[strings.Join(tokens[n:], " ")]; ok {
			return name
		}
	}

	return ""
}

const b2bCityCookie = "b2b_city"
const b2bDeptCookie = "b2b_dept"

// PlaceFilterFromInputs resolves department + city dropdowns. Query params
// win over cookies. Empty defaults to Todos / Todos (all businesses and zones).
func PlaceFilterFromInputs(queryDept, queryCity, rememberedDept, rememberedCity string) (filter, deptVal, cityVal string, all bool) {
	dept := strings.TrimSpace(queryDept)
	city := strings.TrimSpace(queryCity)

	if dept == "" && city == "" {
		dept = strings.TrimSpace(rememberedDept)
		city = strings.TrimSpace(rememberedCity)
	}

	if isCundinamarcaFilter(city) {
		dept = CundinamarcaRegion
		city = PlaceAll
	}

	if isBoyacaFilter(city) {
		dept = BoyacaRegion
		city = PlaceAll
	}

	deptVal = NormalizeDepartment(dept)
	cityVal = NormalizeCityChoice(city)

	if cityVal != PlaceAll && deptVal == PlaceAll {
		if d := DepartmentOfCity(cityVal); d != "" {
			deptVal = d
		}
	}

	if cityVal != PlaceAll && deptVal != PlaceAll {
		allowed := false
		for _, name := range CitiesForDepartment(deptVal) {
			if name == cityVal {
				allowed = true
				break
			}
		}

		if !allowed {
			cityVal = PlaceAll
		}
	}

	filter, all = ResolvePlaceFilter(deptVal, cityVal)

	return filter, deptVal, cityVal, all
}

// CityFilterFromInputs resolves the city dropdown. An explicit query string
// wins. Empty defaults to all businesses (Todos).
func CityFilterFromInputs(queryCity, remembered string) (city string, all bool, cityVal string) {
	filter, _, cityVal, all := PlaceFilterFromInputs("", queryCity, "", remembered)
	if all {
		return "", true, PlaceAll
	}

	return filter, false, cityVal
}
