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

	if len(list[0].Specialties) < 10 {
		t.Fatalf("restaurantes should fan out many specialties, got %d", len(list[0].Specialties))
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

	for _, want := range []string{"restaurantes", "taquerías", "cafeterías", "comida mexicana", "restaurantes colombianos"} {
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
