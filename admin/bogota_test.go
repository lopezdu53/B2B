package admin

import (
	"encoding/json"
	"testing"
)

type geoFC struct {
	Type     string `json:"type"`
	Features []struct {
		Properties struct {
			Nombre    string `json:"nombre"`
			Localidad string `json:"localidad"`
			Codigo    string `json:"codigo"`
			Color     string `json:"color"`
		} `json:"properties"`
	} `json:"features"`
}

func TestBogotaGeoAssets(t *testing.T) {
	raw, err := staticFS.ReadFile("static/geo/bogota-index.json")
	if err != nil {
		t.Fatalf("index: %v", err)
	}

	var idx BogotaIndex
	if err := json.Unmarshal(raw, &idx); err != nil {
		t.Fatalf("index json: %v", err)
	}

	if len(idx.Localidades) != 20 {
		t.Fatalf("localidades in index: got %d want 20", len(idx.Localidades))
	}

	var usaquen int
	for _, loc := range idx.Localidades {
		if loc.Nombre == "" || loc.Color == "" || loc.Codigo == "" {
			t.Fatalf("incomplete localidad %#v", loc)
		}
		if loc.Nombre == "Usaquén" {
			usaquen = len(loc.Barrios)
		}
		if loc.Nombre == "La Candelaria" && len(loc.Barrios) < 10 {
			t.Fatalf("La Candelaria should have barrios, got %d", len(loc.Barrios))
		}
	}

	if usaquen < 200 {
		t.Fatalf("Usaquén barrios: got %d", usaquen)
	}

	locRaw, err := staticFS.ReadFile("static/geo/bogota-localidades.json")
	if err != nil {
		t.Fatalf("localidades: %v", err)
	}

	var locs geoFC
	if err := json.Unmarshal(locRaw, &locs); err != nil {
		t.Fatalf("localidades json: %v", err)
	}

	if len(locs.Features) != 20 {
		t.Fatalf("localidad polygons: got %d want 20", len(locs.Features))
	}

	barRaw, err := staticFS.ReadFile("static/geo/bogota-barrios.json")
	if err != nil {
		t.Fatalf("barrios: %v", err)
	}

	var bars geoFC
	if err := json.Unmarshal(barRaw, &bars); err != nil {
		t.Fatalf("barrios json: %v", err)
	}

	if len(bars.Features) < 3000 {
		t.Fatalf("barrio polygons: got %d, expected thousands", len(bars.Features))
	}

	if bars.Features[0].Properties.Localidad == "" || bars.Features[0].Properties.Nombre == "" {
		t.Fatal("barrio feature missing nombre/localidad")
	}
}

func TestBogotaUrbanLocalidades(t *testing.T) {
	locs := BogotaUrbanLocalidades()
	if len(locs) != 19 {
		t.Fatalf("urban localidades: got %d want 19", len(locs))
	}

	names := map[string]bool{}
	for _, loc := range locs {
		if loc.Codigo == sumapazCodigo {
			t.Fatal("Sumapaz should be excluded from the search dropdown")
		}
		if loc.Nombre == "" {
			t.Fatal("empty localidad name")
		}
		names[loc.Nombre] = true
	}

	for _, want := range []string{"Usaquén", "Chapinero", "Kennedy", "Suba", "La Candelaria"} {
		if !names[want] {
			t.Errorf("missing %q", want)
		}
	}
}

func TestLocalidadSearchHintUsaquen(t *testing.T) {
	hint, ok := LocalidadSearchHint("Usaquén")
	if !ok {
		t.Fatal("missing Usaquén hint")
	}

	if hint.Lat < 4.65 || hint.Lat > 4.82 || hint.Lon > -74.00 || hint.Lon < -74.08 {
		t.Fatalf("Usaquén centroid out of range: %+v", hint)
	}

	if hint.Zoom < 12 || hint.Zoom > 15 {
		t.Fatalf("zoom=%d", hint.Zoom)
	}
}

func TestLocalidadSearchHintAllUrban(t *testing.T) {
	for _, loc := range BogotaUrbanLocalidades() {
		hint, ok := LocalidadSearchHint(loc.Nombre)
		if !ok {
			t.Errorf("missing hint for %q", loc.Nombre)
			continue
		}

		if hint.Lat < 4.2 || hint.Lat > 4.9 || hint.Lon > -73.9 || hint.Lon < -74.3 {
			t.Errorf("%s centroid out of Bogotá: %+v", loc.Nombre, hint)
		}
	}
}

func TestResolveGoogleMapsAPIKeyPrefersEnv(t *testing.T) {
	st := &AppState{GoogleMapsAPIKey: "  env-key  "}
	if got := resolveGoogleMapsAPIKey(st, nil); got != "env-key" {
		t.Fatalf("got %q", got)
	}
}
