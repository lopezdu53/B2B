//nolint:testpackage // renders the embedded templates FS, which is unexported
package admin

import (
	"html/template"
	"io"
	"strings"
	"testing"
)

// TestB2BTemplateRenders ensures the B2B page template (and its partials) parse
// and execute against the data shape the handler provides. This catches template
// syntax / field errors that would otherwise only surface at runtime.
func TestB2BTemplateRenders(t *testing.T) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	advisorCity := "Bogotá"
	advisorID := int64(1)
	bogotaIdx, err := LoadBogotaIndex()
	if err != nil {
		t.Fatalf("bogota index: %v", err)
	}

	advisors := []Advisor{{ID: advisorID, Name: "Ana", City: advisorCity, Email: "ana@example.com", Phone: "300"}}
	zones := []Zone{{ID: 1, Name: "Centro", City: advisorCity, AdvisorID: &advisorID, Color: "#2563eb"}}

	data := map[string]any{
		"CSRFToken":         "test-token",
		"Advisors":          advisors,
		"Zones":             zones,
		"AdvisorsData":      asJSON(advisors),
		"ZonesData":         asJSON(zones),
		"Cities":            []string{"Bogotá", "Medellín"},
		"CityVal":           "Bogotá",
		"BogotaLocalidades": BogotaUrbanLocalidades(),
		"BogotaIndexData":   asJSON(bogotaIdx),
		"SearchRubros":      SearchRubros(),
		"SearchRubrosData":  asJSON(SearchRubros()),
		"PriceSmartData":    asJSON(PriceSmartLocations()),
		"Summary":           &B2BSummary{Total: 10, Clients: 2, Prospects: 6, InProgress: 1, Discarded: 1, Featured: 1, Advisors: 1, Zones: 1},
		"Success":           "",
		"Error":             "",
		// list pages
		"Rows": []businessRow{
			{MapBusiness: MapBusiness{Title: "Rest", City: "Bogotá", Status: "client"}, StatusLabel: "Cliente", StatusClass: "client", AdvisorName: "Ana"},
		},
		"Categories":       []Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 3}},
		"CategoriesData":   asJSON([]Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 3}}),
		"CanManage":        true,
		"IsAdvisor":        false,
		"Count":            1,
		"QVal":             "",
		"StatVal":          "",
		"AdvVal":           "",
		"CatVal":           "",
		"SortVal":          "name",
		"Page":             1,
		"TotalPages":       1,
		"HasPrev":          false,
		"HasNext":          false,
		"PrevURL":          "",
		"NextURL":          "",
		"ExportURL":        "/admin/b2b/negocios/export",
		"EmbedToken":       "1.deadbeef",
		"CanEmbed":         true,
		"GoogleMapsAPIKey": "",
	}

	for _, name := range []string{"b2b.html", "negocios.html", "papelera.html", "asesores.html"} {
		if err := tmpl.ExecuteTemplate(io.Discard, name, data); err != nil {
			t.Fatalf("execute %s: %v", name, err)
		}
	}

	// embed_map.html is the standalone, framable presentation map.
	embedData := map[string]any{
		"Token":             "1.deadbeef",
		"Cities":            []string{"Bogotá", "Medellín"},
		"CityVal":           "Bogotá",
		"Advisors":          advisors,
		"Zones":             zones,
		"Categories":        []Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 3}},
		"AdvisorsData":      asJSON(advisors),
		"ZonesData":         asJSON(zones),
		"CategoriesData":    asJSON([]Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 3}}),
		"PriceSmartData":    asJSON(PriceSmartLocations()),
		"GoogleMapsAPIKey":  "AIza-test",
		"BogotaLocalidades": BogotaUrbanLocalidades(),
		"AssetVersion":      "test",
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "embed_map.html", embedData); err != nil {
		t.Fatalf("execute embed_map.html: %v", err)
	}

	// zonas.html needs zoneRow values.
	zdata := map[string]any{
		"CSRFToken": "t",
		"Advisors":  advisors,
		"Rows":      []zoneRow{{Zone: zones[0], AdvisorName: "Ana"}},
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "zonas.html", zdata); err != nil {
		t.Fatalf("execute zonas.html: %v", err)
	}

	// clientes.html needs tenantRow values.
	cdata := map[string]any{
		"CSRFToken": "t",
		"ActiveID":  int64(1),
		"Rows":      []tenantRow{{Tenant: Tenant{ID: 1, Name: "Acme"}, Admins: 1, Advisors: 3}},
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "clientes.html", cdata); err != nil {
		t.Fatalf("execute clientes.html: %v", err)
	}

	catData := map[string]any{
		"CSRFToken": "t",
		"Rows":      []Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 5}},
	}
	var catBuf strings.Builder
	if err := tmpl.ExecuteTemplate(&catBuf, "categorias.html", catData); err != nil {
		t.Fatalf("execute categorias.html: %v", err)
	}
	catHTML := catBuf.String()
	if strings.Contains(catHTML, "Nueva categoría") || strings.Contains(catHTML, "Crear categoría") || strings.Contains(catHTML, "/delete") {
		t.Fatal("categorias.html should not allow creating or deleting categories")
	}
	if !strings.Contains(catHTML, "Restaurantes") || !strings.Contains(catHTML, "Color e ícono") {
		t.Fatal("categorias.html should list fixed categories and style-only edit")
	}

	// Advisor view hides management chrome; the template must still render.
	data["CanManage"] = false
	data["IsAdvisor"] = true
	data["CanEmbed"] = false
	if err := tmpl.ExecuteTemplate(io.Discard, "b2b.html", data); err != nil {
		t.Fatalf("execute b2b.html as advisor: %v", err)
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "negocios.html", data); err != nil {
		t.Fatalf("execute negocios.html as advisor: %v", err)
	}

	sdata := map[string]any{
		"CSRFToken":         "t",
		"IsSuperadmin":      true,
		"GoogleMapsKeySet":  false,
		"GoogleMapsKeyMask": "",
		"TOTPEnabled":       false,
		"Username":          "admin",
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "settings.html", sdata); err != nil {
		t.Fatalf("execute settings.html: %v", err)
	}
}

func TestB2BSearchFormHasCascadedLocationSelects(t *testing.T) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	idx, err := LoadBogotaIndex()
	if err != nil {
		t.Fatalf("index: %v", err)
	}

	data := map[string]any{
		"CSRFToken":         "t",
		"Advisors":          []Advisor{},
		"Zones":             []Zone{},
		"AdvisorsData":      asJSON([]Advisor{}),
		"ZonesData":         asJSON([]Zone{}),
		"Cities":            []string{"Bogotá", "Medellín"},
		"CityVal":           "Bogotá",
		"BogotaLocalidades": BogotaUrbanLocalidades(),
		"BogotaIndexData":   asJSON(idx),
		"SearchRubros":      SearchRubros(),
		"SearchRubrosData":  asJSON(SearchRubros()),
		"Summary":           &B2BSummary{},
		"CanManage":         true,
		"IsAdvisor":         false,
		"CanEmbed":          false,
		"GoogleMapsAPIKey":  "",
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "b2b.html", data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	html := buf.String()
	for _, needle := range []string{
		`name="category"`,
		`name="specialty"`,
		`Todas las especialidades`,
		`Restaurantes`,
		`SúperMercados`,
		`Hoteles`,
		`taquerías`,
		`areperías`,
		`cevicherías`,
		`comida peruana`,
		`heladerías`,
		`name="city"`,
		`name="localidad"`,
		`name="barrio"`,
		`id="s-city"`,
		`id="s-localidad"`,
		`id="s-barrio"`,
		`Todos los barrios`,
		`Bogotá`,
		`Usaquén`,
		`Chapinero`,
		`Kennedy`,
		`Suba`,
		`name="max_results"`,
		`Rápida (100 negocios)`,
		`Normal (500 negocios)`,
		`Amplia (1000 negocios)`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("search form missing %q", needle)
		}
	}

	if strings.Contains(html, `name="where"`) {
		t.Error("legacy free-text where field should be gone from search form")
	}
}

func TestB2BTemplateHasDrawZoneControls(t *testing.T) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	drawn := []Zone{{ID: 2, Name: "Norte", City: "Bogotá", Color: "#0891b2", Geometry: []byte(`{"type":"Polygon"}`)}}
	data := map[string]any{
		"CSRFToken":         "t",
		"Advisors":          []Advisor{},
		"Zones":             drawn,
		"AdvisorsData":      asJSON([]Advisor{}),
		"ZonesData":         asJSON([]Zone{}),
		"Cities":            []string{"Bogotá", "Medellín"},
		"CityVal":           "Bogotá",
		"BogotaLocalidades": BogotaUrbanLocalidades(),
		"SearchRubros":      SearchRubros(),
		"SearchRubrosData":  asJSON(SearchRubros()),
		"Summary":           &B2BSummary{},
		"CanManage":         true,
		"IsAdvisor":         false,
		"CanEmbed":          false,
		"GoogleMapsAPIKey":  "AIza-test",
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "b2b.html", data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	html := buf.String()
	for _, needle := range []string{
		`id="btn-draw-zone"`,
		`id="btn-draw-undo"`,
		`id="draw-zone-modal"`,
		`id="tog-zones"`,
		`Zonas dibujadas`,
		`libraries=drawing`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("b2b.html missing %q", needle)
		}
	}

	if strings.Contains(html, `id="tog-bar" checked`) {
		t.Error("barrio lines should be off by default")
	}

	if strings.Contains(html, "Asesores y zonas") || strings.Contains(html, "Agregar asesor") {
		t.Error("map page should not include the advisors/zones management section")
	}

	data["CanManage"] = false
	data["IsAdvisor"] = true

	buf.Reset()

	if err := tmpl.ExecuteTemplate(&buf, "b2b.html", data); err != nil {
		t.Fatalf("execute advisor: %v", err)
	}

	if strings.Contains(buf.String(), `id="btn-draw-zone"`) {
		t.Error("advisor view should not offer zone drawing")
	}

	zdata := map[string]any{
		"CSRFToken": "t",
		"Advisors":  []Advisor{},
		"Rows":      []zoneRow{{Zone: drawn[0]}},
	}

	buf.Reset()

	if err := tmpl.ExecuteTemplate(&buf, "zonas.html", zdata); err != nil {
		t.Fatalf("execute zonas.html: %v", err)
	}

	zonas := buf.String()

	if !strings.Contains(zonas, "/admin/b2b#dibujar") || !strings.Contains(zonas, "Dibujada") {
		t.Error("zonas.html should link to map drawing and show drawn badge")
	}
}

func TestB2BTemplateHasLeadQualityAndZoneStats(t *testing.T) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	data := map[string]any{
		"CSRFToken":         "t",
		"Advisors":          []Advisor{},
		"Zones":             []Zone{{ID: 1, Name: "Norte", City: "Bogotá", Color: "#0891b2", Geometry: []byte(`{"type":"Polygon"}`)}},
		"AdvisorsData":      asJSON([]Advisor{}),
		"ZonesData":         asJSON([]Zone{}),
		"Cities":            []string{"Bogotá", "Chía"},
		"CityVal":           "Bogotá",
		"BogotaLocalidades": BogotaUrbanLocalidades(),
		"SearchRubros":      SearchRubros(),
		"SearchRubrosData":  asJSON(SearchRubros()),
		"PriceSmartData":    asJSON(PriceSmartLocations()),
		"Summary":           &B2BSummary{},
		"CanManage":         true,
		"IsAdvisor":         false,
		"CanEmbed":          false,
		"GoogleMapsAPIKey":  "",
		"Rows": []businessRow{{
			MapBusiness: MapBusiness{
				Title: "Rest", City: "Bogotá", Status: "client",
				Website: "rest.com", MapsURL: "https://maps.google.com/?cid=1", Email: "a@b.com",
				Specialty: "Taquerías",
			},
			StatusLabel: "Cliente", StatusClass: "client",
		}},
		"Categories": []Category{},
		"Count":      1,
		"QVal":       "",
		"StatVal":    "",
		"AdvVal":     "",
		"CatVal":     "",
		"SortVal":    "name",
		"Page":       1,
		"TotalPages": 1,
		"ExportURL":  "/admin/b2b/negocios/export",
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "b2b.html", data); err != nil {
		t.Fatalf("execute: %v", err)
	}

	html := buf.String()
	for _, needle := range []string{
		`id="zone-stats"`,
		`id="btn-delete-businesses"`,
		`/admin/b2b/businesses/delete-all`,
		`id="btn-map-full"`,
		`Pantalla completa`,
		`value="featured"`,
		`Destacado`,
		`PRICE_SMART`,
		`Página web`,
		`Ver en Google Maps`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("b2b.html missing %q", needle)
		}
	}

	buf.Reset()
	if err := tmpl.ExecuteTemplate(&buf, "negocios.html", data); err != nil {
		t.Fatalf("execute negocios: %v", err)
	}

	neg := buf.String()
	for _, needle := range []string{
		`id="btn-delete-businesses"`,
		`value="featured"`,
		`target="_blank"`,
	} {
		if !strings.Contains(neg, needle) {
			t.Errorf("negocios.html missing %q", needle)
		}
	}
}
