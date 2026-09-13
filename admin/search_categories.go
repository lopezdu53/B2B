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
	Group   string `json:"group,omitempty"`
}

// SearchRubro is a predefined search family (Restaurantes, Supermercados, Hoteles).
type SearchRubro struct {
	ID          string            `json:"id"`
	Label       string            `json:"label"`
	Specialties []SearchSpecialty `json:"specialties"`
}

func sp(group, label, keyword string) SearchSpecialty {
	return SearchSpecialty{Group: group, Label: label, Keyword: keyword}
}

// SearchRubros is the curated list shown in "Qué buscar".
// "Todas las especialidades" fans out to every keyword in Specialties.
func SearchRubros() []SearchRubro {
	return []SearchRubro{
		{
			ID:    RubroRestaurantes,
			Label: "Restaurantes",
			Specialties: []SearchSpecialty{
				sp("General", "Restaurantes (general)", "restaurantes"),
				sp("General", "Comida internacional", "comida internacional"),
				sp("General", "Comida fusión", "comida fusión"),
				sp("General", "Alta cocina", "alta cocina"),
				sp("General", "Buffets", "buffets"),
				sp("General", "Corrientazos", "corrientazos"),
				sp("General", "Comidas corrientes", "comidas corrientes"),
				sp("General", "Menú ejecutivo", "menú ejecutivo"),

				sp("Comida colombiana", "Restaurantes colombianos", "restaurantes colombianos"),
				sp("Comida colombiana", "Comida típica", "comida típica"),
				sp("Comida colombiana", "Comida paisa", "comida paisa"),
				sp("Comida colombiana", "Comida costeña", "comida costeña"),
				sp("Comida colombiana", "Comida valluna", "comida valluna"),
				sp("Comida colombiana", "Comida santandereana", "comida santandereana"),
				sp("Comida colombiana", "Comida llanera", "comida llanera"),
				sp("Comida colombiana", "Areperías", "areperías"),
				sp("Comida colombiana", "Arepas", "arepas"),
				sp("Comida colombiana", "Empanadas", "empanadas"),
				sp("Comida colombiana", "Lechonerías", "lechonerías"),
				sp("Comida colombiana", "Ajiaco", "ajiaco"),
				sp("Comida colombiana", "Tamales", "tamales"),
				sp("Comida colombiana", "Bandeja paisa", "bandeja paisa"),
				sp("Comida colombiana", "Sancocho", "sancocho"),

				sp("Carnes y asados", "Asaderos", "asaderos"),
				sp("Carnes y asados", "Parrilla", "parrilla"),
				sp("Carnes y asados", "Parrilla argentina", "parrilla argentina"),
				sp("Carnes y asados", "Churrasquería", "churrasquería"),
				sp("Carnes y asados", "Pollerías", "pollerías"),
				sp("Carnes y asados", "Pollo asado", "pollo asado"),
				sp("Carnes y asados", "Carnes", "carnes"),
				sp("Carnes y asados", "Steakhouse", "steakhouse"),
				sp("Carnes y asados", "Barbecue", "barbecue"),

				sp("Mariscos", "Mariscos", "mariscos"),
				sp("Mariscos", "Cevicherías", "cevicherías"),
				sp("Mariscos", "Comida de mar", "comida de mar"),
				sp("Mariscos", "Pescados", "pescados"),

				sp("Comida latina", "Comida mexicana", "comida mexicana"),
				sp("Comida latina", "Taquerías", "taquerías"),
				sp("Comida latina", "Comida peruana", "comida peruana"),
				sp("Comida latina", "Comida venezolana", "comida venezolana"),
				sp("Comida latina", "Comida brasileña", "comida brasileña"),
				sp("Comida latina", "Comida argentina", "comida argentina"),
				sp("Comida latina", "Comida cubana", "comida cubana"),
				sp("Comida latina", "Comida caribeña", "comida caribeña"),

				sp("Internacional", "Comida italiana", "comida italiana"),
				sp("Internacional", "Pasta", "pasta"),
				sp("Internacional", "Comida española", "comida española"),
				sp("Internacional", "Tapas", "tapas"),
				sp("Internacional", "Comida francesa", "comida francesa"),
				sp("Internacional", "Comida japonesa", "comida japonesa"),
				sp("Internacional", "Sushi", "sushi"),
				sp("Internacional", "Ramen", "ramen"),
				sp("Internacional", "Comida china", "comida china"),
				sp("Internacional", "Dim sum", "dim sum"),
				sp("Internacional", "Wok", "wok"),
				sp("Internacional", "Comida coreana", "comida coreana"),
				sp("Internacional", "Comida tailandesa", "comida tailandesa"),
				sp("Internacional", "Comida vietnamita", "comida vietnamita"),
				sp("Internacional", "Comida india", "comida india"),
				sp("Internacional", "Comida árabe", "comida árabe"),
				sp("Internacional", "Shawarma", "shawarma"),
				sp("Internacional", "Kebab", "kebab"),
				sp("Internacional", "Falafel", "falafel"),
				sp("Internacional", "Comida mediterránea", "comida mediterránea"),
				sp("Internacional", "Comida griega", "comida griega"),
				sp("Internacional", "Comida turca", "comida turca"),
				sp("Internacional", "Comida americana", "comida americana"),

				sp("Comida rápida", "Comida rápida", "comida rápida"),
				sp("Comida rápida", "Hamburgueserías", "hamburgueserías"),
				sp("Comida rápida", "Pizzerías", "pizzerías"),
				sp("Comida rápida", "Hot dogs", "hot dogs"),
				sp("Comida rápida", "Perros calientes", "perros calientes"),
				sp("Comida rápida", "Salchipapas", "salchipapas"),
				sp("Comida rápida", "Alitas", "alitas"),
				sp("Comida rápida", "Sándwiches", "sándwiches"),
				sp("Comida rápida", "Wraps", "wraps"),

				sp("Cafés y postres", "Cafeterías", "cafeterías"),
				sp("Cafés y postres", "Café", "café"),
				sp("Cafés y postres", "Panaderías", "panaderías"),
				sp("Cafés y postres", "Pastelerías", "pastelerías"),
				sp("Cafés y postres", "Heladerías", "heladerías"),
				sp("Cafés y postres", "Postres", "postres"),
				sp("Cafés y postres", "Crepes", "crepes"),
				sp("Cafés y postres", "Waffles", "waffles"),
				sp("Cafés y postres", "Chocolate", "chocolate"),
				sp("Cafés y postres", "Onces", "onces"),
				sp("Cafés y postres", "Brunch", "brunch"),
				sp("Cafés y postres", "Desayunos", "desayunos"),
				sp("Cafés y postres", "Jugos naturales", "jugos naturales"),
				sp("Cafés y postres", "Fruterías", "fruterías"),
				sp("Cafés y postres", "Açaí", "açaí"),

				sp("Saludable", "Comida vegetariana", "comida vegetariana"),
				sp("Saludable", "Restaurantes veganos", "restaurantes veganos"),
				sp("Saludable", "Comida saludable", "comida saludable"),
				sp("Saludable", "Poke", "poke"),
				sp("Saludable", "Bowls", "bowls"),
			},
		},
		{
			ID:    RubroSupermercados,
			Label: "SúperMercados",
			Specialties: []SearchSpecialty{
				sp("Supermercados", "Supermercados (general)", "supermercados"),
				sp("Supermercados", "Hipermercados", "hipermercados"),
				sp("Supermercados", "Minimercados", "minimercados"),
				sp("Supermercados", "Minimarkets", "minimarkets"),
				sp("Supermercados", "Tiendas de barrio", "tiendas de barrio"),
				sp("Supermercados", "Tiendas de conveniencia", "tiendas de conveniencia"),
				sp("Supermercados", "Hard discount", "hard discount"),
				sp("Supermercados", "Mayoristas", "mayoristas"),
				sp("Supermercados", "Cash and carry", "cash and carry"),
				sp("Mercados", "Plazas de mercado", "plazas de mercado"),
				sp("Mercados", "Mercados", "mercados"),
				sp("Mercados", "Graneros", "graneros"),
				sp("Mercados", "Fruver", "fruver"),
				sp("Mercados", "Carnicerías", "carnicerías"),
				sp("Mercados", "Pescaderías", "pescaderías"),
				sp("Mercados", "Licorerías", "licorerías"),
			},
		},
		{
			ID:    RubroHoteles,
			Label: "Hoteles",
			Specialties: []SearchSpecialty{
				sp("Alojamiento", "Hoteles (general)", "hoteles"),
				sp("Alojamiento", "Hoteles boutique", "hoteles boutique"),
				sp("Alojamiento", "Hoteles de lujo", "hoteles de lujo"),
				sp("Alojamiento", "Hoteles económicos", "hoteles económicos"),
				sp("Alojamiento", "Hostales", "hostales"),
				sp("Alojamiento", "Hosterías", "hosterías"),
				sp("Alojamiento", "Apartahoteles", "apartahoteles"),
				sp("Alojamiento", "Hospedajes", "hospedajes"),
				sp("Alojamiento", "Residencias", "residencias"),
				sp("Alojamiento", "Bed and breakfast", "bed and breakfast"),
				sp("Alojamiento", "Posadas", "posadas"),
				sp("Alojamiento", "Apartamentos turísticos", "apartamentos turísticos"),
				sp("Alojamiento", "Cabañas", "cabañas"),
				sp("Alojamiento", "Glamping", "glamping"),
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
		seen := make(map[string]struct{}, len(rubro.Specialties))
		for _, s := range rubro.Specialties {
			k := strings.TrimSpace(s.Keyword)
			if k == "" {
				continue
			}
			if _, ok := seen[k]; ok {
				continue
			}
			seen[k] = struct{}{}
			out = append(out, k)
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
