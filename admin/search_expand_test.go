package admin

import (
	"strings"
	"testing"
)

func TestResolveSearchPlacesFansOutBogotaLocalidades(t *testing.T) {
	t.Parallel()

	places := ResolveSearchPlaces("Bogotá", "", "")
	urban := BogotaUrbanLocalidades()
	if len(places) != len(urban) || len(places) != 19 {
		t.Fatalf("places=%d urban=%d", len(places), len(urban))
	}

	seen := map[string]bool{}
	for _, p := range places {
		if p.City != DefaultCity || p.Localidad == "" || p.Barrio != "" {
			t.Fatalf("unexpected place %#v", p)
		}
		if seen[p.Localidad] {
			t.Fatalf("duplicate localidad %q", p.Localidad)
		}
		seen[p.Localidad] = true
	}

	for _, want := range []string{"Usaquén", "Chapinero", "Kennedy", "Suba", "Bosa"} {
		if !seen[want] {
			t.Errorf("missing %q", want)
		}
	}
}

func TestResolveSearchPlacesKeepsExplicitLocalidad(t *testing.T) {
	t.Parallel()

	places := ResolveSearchPlaces("Bogotá", "Usaquén", "Cedritos")
	if len(places) != 1 {
		t.Fatalf("got %d places", len(places))
	}

	if places[0] != (SearchPlace{City: "Bogotá", Localidad: "Usaquén", Barrio: "Cedritos"}) {
		t.Fatalf("got %#v", places[0])
	}
}

func TestResolveSearchPlacesOtherCityIsSingleJob(t *testing.T) {
	t.Parallel()

	places := ResolveSearchPlaces("Medellín", "", "")
	if len(places) != 1 || places[0].City != "Medellín" || places[0].Localidad != "" {
		t.Fatalf("got %#v", places)
	}
}

func TestExpandSearchJobsDefaultCoversBogota(t *testing.T) {
	t.Parallel()

	plan := ExpandSearchJobs(RubroRestaurantes, []string{"restaurantes"}, "Bogotá", "", "", "")
	if plan.Places != 19 || plan.Terms != 1 || len(plan.Jobs) != 19 {
		t.Fatalf("plan=%+v jobs=%d", plan, len(plan.Jobs))
	}

	var usaquen SearchJobSpec
	for _, job := range plan.Jobs {
		if strings.Contains(job.Keyword, "Usaquén") {
			usaquen = job
			break
		}
	}

	if usaquen.Keyword != "restaurantes en Usaquén, Bogotá" {
		t.Fatalf("usaquén keyword = %q", usaquen.Keyword)
	}

	if usaquen.Geo == "" || usaquen.Zoom < 12 || usaquen.Zoom > 16 {
		t.Fatalf("usaquén viewport geo=%q zoom=%d", usaquen.Geo, usaquen.Zoom)
	}

	if !strings.Contains(usaquen.Geo, "4.") || !strings.Contains(usaquen.Geo, "-74.") {
		t.Fatalf("usaquén geo should be in Bogotá, got %q", usaquen.Geo)
	}
}

func TestExpandSearchJobsDoesNotMultiplySpecialtyAndLocalidad(t *testing.T) {
	t.Parallel()

	terms, _, ok := ResolveSearchTerms(RubroRestaurantes, "all")
	if !ok || len(terms) < 40 {
		t.Fatalf("terms=%d ok=%v", len(terms), ok)
	}

	plan := ExpandSearchJobs(RubroRestaurantes, terms, "Bogotá", "", "", "")
	if plan.Terms != 1 {
		t.Fatalf("should keep the general term only, terms=%d", plan.Terms)
	}

	if len(plan.Jobs) != 19 {
		t.Fatalf("jobs=%d want 19 locality jobs", len(plan.Jobs))
	}

	if !strings.HasPrefix(plan.Jobs[0].Keyword, "restaurantes en ") {
		t.Fatalf("first job should be the general rubro, got %q", plan.Jobs[0].Keyword)
	}
}

func TestExpandSearchJobsSpecialtyFanOutInOneLocalidad(t *testing.T) {
	t.Parallel()

	terms, _, ok := ResolveSearchTerms(RubroRestaurantes, "all")
	if !ok {
		t.Fatal("resolve")
	}

	plan := ExpandSearchJobs(RubroRestaurantes, terms, "Bogotá", "Chapinero", "", "")
	if plan.Places != 1 {
		t.Fatalf("places=%d", plan.Places)
	}

	if len(plan.Jobs) != len(terms) {
		t.Fatalf("jobs=%d terms=%d (cap is %d)", len(plan.Jobs), len(terms), maxB2BSearchJobs)
	}

	if !strings.Contains(plan.Jobs[0].Keyword, "Chapinero") {
		t.Fatalf("keyword=%q", plan.Jobs[0].Keyword)
	}
}

func TestSearchMapHintOtherCityHasNoViewport(t *testing.T) {
	t.Parallel()

	geo, zoom := SearchMapHint("Medellín", "", "")
	if geo != "" || zoom != 0 {
		t.Fatalf("geo=%q zoom=%d", geo, zoom)
	}
}

func TestSearchQueuedMessageLocalidades(t *testing.T) {
	t.Parallel()

	msg := searchQueuedMessage(
		[]string{"a", "b"},
		[]string{"restaurantes en Usaquén, Bogotá", "restaurantes en Suba, Bogotá"},
		"Restaurantes",
		"Bogotá",
		SearchPlan{Places: 19, Terms: 1},
	)
	if !strings.Contains(msg, "localidad") {
		t.Fatalf("msg=%q", msg)
	}
}
