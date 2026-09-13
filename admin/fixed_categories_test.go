package admin

import "testing"

func TestCanonicalFixedCategory(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"Restaurantes":  "Restaurantes",
		"restaurante":   "Restaurantes",
		"Supermercados": "SúperMercados",
		"SúperMercados": "SúperMercados",
		"Hoteles":       "Hoteles",
		"hotel":         "Hoteles",
		"Pizzerías":     "",
	}

	for in, want := range cases {
		if got := CanonicalFixedCategory(in); got != want {
			t.Errorf("CanonicalFixedCategory(%q)=%q want %q", in, got, want)
		}
	}
}

func TestInferFixedCategory(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"Hamburgueserías":   "Restaurantes",
		"Pizzerías":         "Restaurantes",
		"Cafeterías":        "Restaurantes",
		"Hostales":          "Hoteles",
		"Hoteles boutique":  "Hoteles",
		"Minimercados":      "SúperMercados",
		"Tiendas de barrio": "SúperMercados",
		"Restaurantes":      "Restaurantes",
	}

	for in, want := range cases {
		if got := InferFixedCategory(in); got != want {
			t.Errorf("InferFixedCategory(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFilterFixedCategoriesOrderAndDropsExtras(t *testing.T) {
	t.Parallel()

	in := []Category{
		{ID: 9, Name: "Pizzerías"},
		{ID: 3, Name: "Hoteles"},
		{ID: 1, Name: "Restaurantes"},
		{ID: 2, Name: "Supermercados"},
	}

	out := FilterFixedCategories(in)
	if len(out) != 3 {
		t.Fatalf("got %d", len(out))
	}

	if out[0].Name != "Restaurantes" || out[1].Name != "SúperMercados" || out[2].Name != "Hoteles" {
		t.Fatalf("order = %#v", out)
	}

	if out[1].ID != 2 {
		t.Fatalf("supermercados id = %d", out[1].ID)
	}
}
