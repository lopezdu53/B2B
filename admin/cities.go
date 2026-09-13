package admin

import (
	"strings"
	"unicode"
)

// DefaultCity is the city selected on first load of map and list views.
const DefaultCity = "Bogotá"

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
}

// CityList returns a copy of the predefined city dropdown (Bogotá first).
func CityList() []string {
	out := make([]string, len(PredefinedCities))
	copy(out, PredefinedCities)

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

// ResolveCityFilter interprets a city query parameter.
// "all" / "todas" / "*" means no city filter. Empty defaults to Bogotá.
func ResolveCityFilter(raw string) (city string, all bool) {
	raw = strings.TrimSpace(raw)
	switch strings.ToLower(raw) {
	case "all", "todas", "*":
		return "", true
	case "":
		return DefaultCity, false
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
