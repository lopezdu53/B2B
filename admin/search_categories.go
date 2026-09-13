package admin

import "strings"

const (
	RubroRestaurantes  = "restaurantes"
	RubroSupermercados = "supermercados"
	RubroHoteles       = "hoteles"
)

// SearchSpecialty is one Google Maps query under a parent rubro.
type SearchSpecialty struct {
	Label   string `json:"label"`
	Keyword string `json:"keyword"`
}

// SearchRubro is a predefined search family (Restaurantes, Supermercados, Hoteles).
type SearchRubro struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Specialties []SearchSpecialty `json:"specialties"`
}

// SearchRubros is the curated list shown in "Qué buscar".
// "Todas las especialidades" fans out to every keyword in Specialties.
func SearchRubros() []SearchRubro {
	return []SearchRubro{
		{
			ID:    RubroRestaurantes,
			Label: "Restaurantes",
			Specialties: []SearchSpecialty{
				{Label: "Restaurantes (general)", Keyword: "restaurantes"},
				{Label: "Cafeterías", Keyword: "cafeterías"},
				{Label: "Taquerías", Keyword: "taquerías"},
				{Label: "Comida mexicana", Keyword: "comida mexicana"},
				{Label: "Restaurantes colombianos", Keyword: "restaurantes colombianos"},
				{Label: "Asaderos", Keyword: "asaderos"},
				{Label: "Parrilla", Keyword: "parrilla"},
				{Label: "Comida rápida", Keyword: "comida rápida"},
				{Label: "Hamburgueserías", Keyword: "hamburgueserías"},
				{Label: "Pizzerías", Keyword: "pizzerías"},
				{Label: "Sushi", Keyword: "sushi"},
				{Label: "Comida china", Keyword: "comida china"},
				{Label: "Comida árabe", Keyword: "comida árabe"},
				{Label: "Panaderías", Keyword: "panaderías"},
			},
		},
		{
			ID:    RubroSupermercados,
			Label: "SúperMercados",
			Specialties: []SearchSpecialty{
				{Label: "Supermercados (general)", Keyword: "supermercados"},
				{Label: "Hipermercados", Keyword: "hipermercados"},
				{Label: "Minimercados", Keyword: "minimercados"},
				{Label: "Tiendas de barrio", Keyword: "tiendas de barrio"},
				{Label: "Hard discount", Keyword: "hard discount"},
			},
		},
		{
			ID:    RubroHoteles,
			Label: "Hoteles",
			Specialties: []SearchSpecialty{
				{Label: "Hoteles (general)", Keyword: "hoteles"},
				{Label: "Hostales", Keyword: "hostales"},
				{Label: "Apartahoteles", Keyword: "apartahoteles"},
				{Label: "Hospedajes", Keyword: "hospedajes"},
			},
		},
	}
}

// FindSearchRubro returns the rubro by id, or nil.
func FindSearchRubro(id string) *SearchRubro {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, r := range SearchRubros() {
		if r.ID == id {
			cp := r
			return &cp
		}
	}

	return nil
}

// ResolveSearchTerms maps the form fields to one or more Google Maps keywords.
// Empty specialty / "all" fans out to every specialty of the rubro.
func ResolveSearchTerms(rubroID, specialtyKeyword string) ([]string, string, bool) {
	rubro := FindSearchRubro(rubroID)
	if rubro == nil {
		return nil, "", false
	}

	specialtyKeyword = strings.TrimSpace(specialtyKeyword)
	switch strings.ToLower(specialtyKeyword) {
	case "", "all", "todas", "*":
		out := make([]string, 0, len(rubro.Specialties))
		for _, s := range rubro.Specialties {
			if k := strings.TrimSpace(s.Keyword); k != "" {
				out = append(out, k)
			}
		}

		return out, rubro.Label, true
	}

	for _, s := range rubro.Specialties {
		if s.Keyword == specialtyKeyword {
			return []string{s.Keyword}, rubro.Label, true
		}
	}

	return nil, rubro.Label, false
}
