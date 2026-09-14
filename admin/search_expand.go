package admin

import (
	"fmt"
	"strings"
)

// SearchPlace is one geographic target for a B2B scrape job.
type SearchPlace struct {
	City      string
	Localidad string
	Barrio    string
}

// SearchJobSpec is one keyword + map viewport ready to enqueue.
type SearchJobSpec struct {
	Keyword   string
	Specialty string
	Geo       string
	Zoom      int
}

// SearchPlan is the expanded set of scrape jobs for a dashboard search.
type SearchPlan struct {
	Jobs   []SearchJobSpec
	Places int
	Terms  int
}

// ResolveSearchPlaces turns the cascaded city / localidad / barrio fields
// into one or more scrape targets. An empty localidad in Bogotá fans out to
// every urban localidad: a single "restaurantes en Bogotá" job only returns
// Google Maps' city-wide ~120 popular pins.
func ResolveSearchPlaces(city, localidad, barrio string) []SearchPlace {
	city = strings.TrimSpace(city)
	if canon := CanonicalCity(city); canon != "" {
		city = canon
	}

	if city == "" {
		city = DefaultCity
	}

	localidad = strings.TrimSpace(localidad)
	barrio = strings.TrimSpace(barrio)

	if barrio != "" || localidad != "" {
		return []SearchPlace{{City: city, Localidad: localidad, Barrio: barrio}}
	}

	if city == DefaultCity {
		locs := BogotaUrbanLocalidades()
		out := make([]SearchPlace, 0, len(locs))

		for _, loc := range locs {
			out = append(out, SearchPlace{City: DefaultCity, Localidad: loc.Nombre})
		}

		if len(out) > 0 {
			return out
		}
	}

	return []SearchPlace{{City: city}}
}

// ExpandSearchJobs builds the scrape queue for a dashboard search.
// Specialty fan-out and locality fan-out are not multiplied together: if
// both would explode, locality coverage keeps the first (general) term.
func ExpandSearchJobs(rubroID string, terms []string, city, localidad, barrio, where string) SearchPlan {
	if len(terms) == 0 {
		return SearchPlan{}
	}

	places := ResolveSearchPlaces(city, localidad, barrio)
	if len(places) == 0 {
		places = []SearchPlace{{City: DefaultCity}}
	}

	if len(terms) > 1 && len(places) > 1 {
		terms = terms[:1]
	}

	plan := SearchPlan{Places: len(places), Terms: len(terms)}
	plan.Jobs = make([]SearchJobSpec, 0, len(terms)*len(places))

	for _, term := range terms {
		spec := SpecialtyLabelForKeyword(rubroID, term)
		if spec == "" {
			spec = term
		}

		for _, place := range places {
			geo, zoom := SearchMapHint(place.City, place.Localidad, place.Barrio)
			plan.Jobs = append(plan.Jobs, SearchJobSpec{
				Keyword:   SearchKeyword(term, place.City, place.Localidad, place.Barrio, where),
				Specialty: spec,
				Geo:       geo,
				Zoom:      zoom,
			})

			if len(plan.Jobs) >= maxB2BSearchJobs {
				return plan
			}
		}
	}

	return plan
}

// SearchMapHint returns a lat,lon + zoom so Google Maps scrapes a localidad
// viewport instead of the whole city ranking.
func SearchMapHint(city, localidad, barrio string) (geo string, zoom int) {
	city = strings.TrimSpace(city)
	if canon := CanonicalCity(city); canon != "" {
		city = canon
	}

	localidad = strings.TrimSpace(localidad)
	if localidad == "" && city != DefaultCity && city != "" {
		return "", 0
	}

	name := localidad
	if name == "" {
		name = city
	}

	hint, ok := LocalidadSearchHint(name)
	if !ok {
		return "", 0
	}

	geo = fmt.Sprintf("%.5f,%.5f", hint.Lat, hint.Lon)
	zoom = hint.Zoom
	if strings.TrimSpace(barrio) != "" && zoom < 15 {
		zoom = 15
	}

	return geo, zoom
}
