package admin

import "testing"

func TestSearchRubrosHasThreeParents(t *testing.T) {
	t.Parallel()

	list := SearchRubros()
	if len(list) != 3 {
		t.Fatalf("got %d rubros", len(list))
	}

	if list[0].ID != RubroRestaurantes || list[0].Label != "Restaurantes" {
		t.Fatalf("first rubro = %+v", list[0])
	}

	if list[1].Label != "SúperMercados" {
		t.Fatalf("supermercados label = %q", list[1].Label)
	}

	if len(list[0].Specialties) < 40 {
		t.Fatalf("restaurantes should list many specialties, got %d", len(list[0].Specialties))
	}

	seen := map[string]struct{}{}
	for _, s := range list[0].Specialties {
		if s.Keyword == "" || s.Label == "" || s.Group == "" {
			t.Fatalf("incomplete specialty %#v", s)
		}
		if _, ok := seen[s.Keyword]; ok {
			t.Fatalf("duplicate keyword %q", s.Keyword)
		}
		seen[s.Keyword] = struct{}{}
	}
}

func TestResolveSearchTermsFanOut(t *testing.T) {
	t.Parallel()

	terms, label, ok := ResolveSearchTerms("restaurantes", "")
	if !ok || label != "Restaurantes" {
		t.Fatalf("ok=%v label=%q", ok, label)
	}

	if len(terms) != len(FindSearchRubro(RubroRestaurantes).Specialties) {
		t.Fatalf("fan-out count = %d", len(terms))
	}

	seen := map[string]bool{}
	for _, k := range terms {
		if k == "" || seen[k] {
			t.Fatalf("bad keyword %q", k)
		}
		seen[k] = true
	}

	for _, want := range []string{
		"restaurantes", "taquerías", "cafeterías", "comida mexicana",
		"restaurantes colombianos", "areperías", "cevicherías",
		"comida peruana", "comida vegetariana", "heladerías", "ramen",
	} {
		if !seen[want] {
			t.Errorf("missing %q", want)
		}
	}
}

func TestResolveSearchTermsOneSpecialty(t *testing.T) {
	t.Parallel()

	terms, label, ok := ResolveSearchTerms("restaurantes", "taquerías")
	if !ok || label != "Restaurantes" {
		t.Fatalf("ok=%v label=%q", ok, label)
	}

	if len(terms) != 1 || terms[0] != "taquerías" {
		t.Fatalf("got %v", terms)
	}
}

func TestResolveSearchTermsRejectsUnknown(t *testing.T) {
	t.Parallel()

	if _, _, ok := ResolveSearchTerms("farmacias", ""); ok {
		t.Fatal("unknown rubro should fail")
	}

	if _, _, ok := ResolveSearchTerms("restaurantes", "no-existe"); ok {
		t.Fatal("unknown specialty should fail")
	}
}

func TestSpecialtyLabelForKeyword(t *testing.T) {
	t.Parallel()

	if got := SpecialtyLabelForKeyword(RubroRestaurantes, "taquerías"); got != "Taquerías" {
		t.Fatalf("got %q", got)
	}

	if got := SpecialtyLabelForKeyword(RubroHoteles, "hostales"); got != "Hostales" {
		t.Fatalf("got %q", got)
	}

	if got := SpecialtyLabelForKeyword("", "no-existe"); got != "no-existe" {
		t.Fatalf("fallback got %q", got)
	}
}

func TestResolveSearchTermsHotelesAndSupermercados(t *testing.T) {
	t.Parallel()

	terms, label, ok := ResolveSearchTerms("hoteles", "hostales")
	if !ok || label != "Hoteles" || len(terms) != 1 || terms[0] != "hostales" {
		t.Fatalf("hoteles: %v %q %v", terms, label, ok)
	}

	terms, label, ok = ResolveSearchTerms("supermercados", "all")
	if !ok || label != "SúperMercados" || len(terms) < 4 {
		t.Fatalf("supermercados: %v %q %v", terms, label, ok)
	}
}
