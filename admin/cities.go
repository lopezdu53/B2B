package admin

import (
	"strings"
	"unicode"
)

// DefaultCity is the urban capital used for Bogotá localidad matching.
const DefaultCity = "Bogotá"

// CundinamarcaRegion is the default map/list filter: Bogotá D.C. plus the
// surrounding municipalities (Chía, La Calera, Cajicá, Soacha, …).
const CundinamarcaRegion = "Cundinamarca"

// PredefinedCities is the filter list. Bogotá is first; the rest are major
// Colombian cities so the dropdown does not depend on whatever a scrape stored.
var PredefinedCities = []string{
	"Bogotá",
	"Medellín",
	"Cali",
	"Barranquilla",
	"Cartagena",
	"Bucaramanga",
	"Pereira",
	"Cúcuta",
	"Santa Marta",
	"Ibagué",
	"Manizales",
	"Villavicencio",
	"Pasto",
	"Neiva",
	"Armenia",
	"Montería",
	"Valledupar",
	"Sincelejo",
	"Popayán",
	"Tunja",
	"Soacha",
	"Chía",
	"La Calera",
	"Cajicá",
	"Cota",
	"Zipaquirá",
	"Facatativá",
	"Fusagasugá",
	"Girardot",
	"Riohacha",
	"Quibdó",
	"Florencia",
	"Yopal",
}

var cityByKey map[string]string

func init() {
	cityByKey = make(map[string]string, len(PredefinedCities)+8)
	for _, name := range PredefinedCities {
		cityByKey[cityKey(name)] = name
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
	cityByKey[cityKey(CundinamarcaRegion)] = CundinamarcaRegion
	cityByKey[cityKey("cundinamarca colombia")] = CundinamarcaRegion

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

// CityList returns the city dropdown with Cundinamarca first (the default
// regional view), then Bogotá and the rest of the predefined cities.
func CityList() []string {
	out := make([]string, 0, len(PredefinedCities)+1)
	out = append(out, CundinamarcaRegion)
	out = append(out, PredefinedCities...)

	return out
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

// ResolveCityFilter interprets a city query parameter.
// "all" / "todas" / "*" means no city filter. Empty defaults to Cundinamarca.
func ResolveCityFilter(raw string) (city string, all bool) {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "all", "todas", "*":
		return "", true
	case "":
		return CundinamarcaRegion, false
	}

	if c := CanonicalCity(raw); c != "" {
		return c, false
	}

	return raw, false
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
// Searches in Bogotá, Chía or La Calera keep the Cundinamarca regional view
// so one town does not hide the rest of the department.
func InferCityFromSearch(what, where string) string {
	c := inferCityFromText(where)
	if c == "" {
		c = inferCityFromText(what)
	}

	if InCundinamarcaRegion(c) {
		return CundinamarcaRegion
	}

	return c
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

// CityFilterFromInputs resolves the city dropdown. An explicit query string
// wins. A remembered Chía/Bogotá/La Calera cookie is widened to Cundinamarca
// so the map shows the whole department instead of a single town.
func CityFilterFromInputs(queryCity, remembered string) (city string, all bool, cityVal string) {
	explicit := strings.TrimSpace(queryCity)
	raw := explicit
	if raw == "" {
		raw = strings.TrimSpace(remembered)
		if raw == "" || InCundinamarcaRegion(raw) {
			raw = CundinamarcaRegion
		}
	}

	city, all = ResolveCityFilter(raw)
	cityVal = city
	if all {
		cityVal = "all"
	}

	if cityVal == "" && !all {
		cityVal = CundinamarcaRegion
	}

	return city, all, cityVal
}
