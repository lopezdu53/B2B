//nolint:testpackage // renders the embedded templates FS, which is unexported
package admin

import (
	"html/template"
	"io"
	"strings"
	"testing"
	"time"
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
		"Cities":            []string{"Bogotá", "Tunja"},
		"Departments":       Departments(),
		"DepartmentsData":   asJSON(Departments()),
		"DeptVal":           PlaceAll,
		"CityVal":           PlaceAll,
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
		"Visits": []ShareVisit{{
			Kind: ShareKindZone, TargetName: "Centro", IP: "190.25.1.10",
			Country: "Colombia", CountryISO: "CO", City: "Bogotá",
			Timezone: "America/Bogota", VisitedAt: time.Date(2026, 9, 19, 16, 5, 0, 0, time.UTC),
		}},
	}

	for _, name := range []string{"b2b.html", "negocios.html", "papelera.html", "asesores.html", "visitas.html"} {
		if err := tmpl.ExecuteTemplate(io.Discard, name, data); err != nil {
			t.Fatalf("execute %s: %v", name, err)
		}
	}

	// embed_map.html is the standalone, framable presentation map.
	embedData := map[string]any{
		"Token":             "1.deadbeef",
		"Cities":            []string{"Bogotá", "Tunja"},
		"Departments":       Departments(),
		"DepartmentsData":   asJSON(Departments()),
		"DeptVal":           PlaceAll,
		"CityVal":           PlaceAll,
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

	embedZoneData := map[string]any{
		"Token":            "z.1.2.deadbeef",
		"Zone":             zones[0],
		"ZoneData":         asJSON(zones),
		"Visitor": ShareVisit{
			IP: "190.25.1.10", Country: "Colombia", CountryISO: "CO", City: "Bogotá",
			Timezone: "America/Bogota", VisitedAt: time.Date(2026, 9, 19, 16, 5, 0, 0, time.UTC),
		},
		"GoogleMapsAPIKey": "AIza-test",
		"AssetVersion":     "test",
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "embed_zona.html", embedZoneData); err != nil {
		t.Fatalf("execute embed_zona.html: %v", err)
	}

	// zonas.html needs zoneRow values.
	zdata := map[string]any{
		"CSRFToken": "t",
		"Advisors":  advisors,
		"Rows":      []zoneRow{{Zone: zones[0], AdvisorName: "Ana", ShareURL: "https://example.com/embed/zona?t=z.1.1.abc"}},
		"Groups": []zoneGroupRow{{
			ZoneGroup: ZoneGroup{ID: 1, Name: "Norte", ZoneIDs: []int64{1}, Zones: zones},
			ShareURL:  "https://example.com/embed/grupo?t=g.1.1.abc",
		}},
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "zonas.html", zdata); err != nil {
		t.Fatalf("execute zonas.html: %v", err)
	}

	embedGroupData := map[string]any{
		"Token":            "g.1.1.deadbeef",
		"Group":            ZoneGroup{ID: 1, Name: "Norte"},
		"ZoneData":         asJSON(zones),
		"Visitor": ShareVisit{
			IP: "190.25.1.10", Country: "Colombia", CountryISO: "CO", City: "Chía",
			Timezone: "America/Bogota", VisitedAt: time.Date(2026, 9, 19, 16, 5, 0, 0, time.UTC),
		},
		"GoogleMapsAPIKey": "AIza-test",
		"AssetVersion":     "test",
	}
	if err := tmpl.ExecuteTemplate(io.Discard, "embed_grupo.html", embedGroupData); err != nil {
		t.Fatalf("execute embed_grupo.html: %v", err)
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

func TestVisitasPageAndEmbedVisitorChip(t *testing.T) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	visit := ShareVisit{
		Kind: ShareKindGroup, TargetName: "Norte unido", IP: "190.25.1.10",
		Country: "Colombia", CountryISO: "CO", City: "Chía",
		Timezone: "America/Bogota", VisitedAt: time.Date(2026, 9, 19, 16, 5, 0, 0, time.UTC),
	}

	var buf strings.Builder
	if err := tmpl.ExecuteTemplate(&buf, "visitas.html", map[string]any{
		"CSRFToken": "t",
		"CanManage": true,
		"Visits":    []ShareVisit{visit},
	}); err != nil {
		t.Fatalf("execute visitas.html: %v", err)
	}

	html := buf.String()
	for _, needle := range []string{"Visitas de mapas públicos", "190.25.1.10", "Chía", "flagcdn.com/24x18/co.png", "/admin/b2b/visitas"} {
		if !strings.Contains(html, needle) {
			t.Errorf("visitas.html missing %q", needle)
		}
	}

	buf.Reset()
	if err := tmpl.ExecuteTemplate(&buf, "embed_zona.html", map[string]any{
		"Token": "z.1.1.abc", "Zone": Zone{Name: "Centro"}, "ZoneData": asJSON([]Zone{}),
		"Visitor": visit, "AssetVersion": "t",
	}); err != nil {
		t.Fatalf("execute embed_zona.html: %v", err)
	}

	embed := buf.String()
	if !strings.Contains(embed, `class="visitor"`) || !strings.Contains(embed, "Chía") || !strings.Contains(embed, "septiembre") {
		t.Fatalf("embed visitor chip missing: %s", embed[strings.Index(embed, "topbar"):strings.Index(embed, "topbar")+800])
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
		"Cities":            []string{"Bogotá", "Tunja"},
		"Departments":       Departments(),
		"DepartmentsData":   asJSON(Departments()),
		"DeptVal":           PlaceAll,
		"CityVal":           PlaceAll,
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
		`name="what"`,
		`id="s-what"`,
		`name="where"`,
		`id="s-where"`,
		`name="max_depth"`,
		`Usaquén, Bogotá`,
		`Chía`,
		`La Calera`,
		`Cundinamarca`,
		`Boyacá`,
		`id="f-dept"`,
		`id="f-city"`,
		`>Todos</option>`,
		`persistCity`,
		`maybeAdoptSearchCity`,
		`cityFromKeyword`,
		`Rápida (~pocos)`,
		`value="10" selected`,
		`Amplia (~muchos)`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("search form missing %q", needle)
		}
	}

	for _, gone := range []string{
		`name="specialty"`,
		`id="s-category"`,
		`id="s-localidad"`,
		`id="s-barrio"`,
		`name="max_results"`,
		`Todas las especialidades`,
		`una búsqueda por cada localidad`,
	} {
		if strings.Contains(html, gone) {
			t.Errorf("search form should not include %q", gone)
		}
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
		"Cities":            []string{"Bogotá", "Tunja"},
		"Departments":       Departments(),
		"DepartmentsData":   asJSON(Departments()),
		"DeptVal":           PlaceAll,
		"CityVal":           PlaceAll,
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
		`id="tog-zones"`,
		`Zonas dibujadas`,
		`id="f-rating"`,
		`2 a 3`,
		`4 a 5`,
		`id="f-dept"`,
		`Boyacá`,
		`>Todos</option>`,
		`fillCityOptions`,
		`department:`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("b2b.html missing %q", needle)
		}
	}

	for _, needle := range []string{
		`id="draw-toolbar"`,
		`id="btn-draw-finish"`,
		`id="btn-draw-undo"`,
		`id="btn-draw-cancel"`,
		`id="btn-draw-streets"`,
		`id="btn-draw-straight"`,
		`Seguir calles`,
		`Líneas rectas`,
		`dibujar-recto`,
		`-zona-`,
		`editingZone`,
		`/admin/b2b/zones/`,
		`class="job-remove"`,
		`/admin/b2b/jobs/`,
	} {
		if !strings.Contains(html, needle) {
			t.Errorf("b2b.html missing %q", needle)
		}
	}

	for _, gone := range []string{
		`id="btn-draw-zone"`,
		`id="btn-delete-businesses"`,
		`id="btn-reset-leads"`,
	} {
		if strings.Contains(html, gone) {
			t.Errorf("map page should not include %q", gone)
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

	advisorHTML := buf.String()
	if strings.Contains(advisorHTML, `id="btn-draw-zone"`) || strings.Contains(advisorHTML, `id="draw-toolbar"`) {
		t.Error("advisor view should not offer zone drawing")
	}

	zdata := map[string]any{
		"CSRFToken": "t",
		"Advisors":  []Advisor{},
		"Rows":      []zoneRow{{Zone: drawn[0], ShareURL: "/embed/zona?t=z.1.1.abc"}},
		"Groups": []zoneGroupRow{{
			ZoneGroup: ZoneGroup{
				ID:      8,
				Name:    "Norte unido",
				ZoneIDs: []int64{2},
				Zones:   drawn,
			},
			ShareURL: "/embed/grupo?t=g.1.8.abc",
		}},
	}

	buf.Reset()

	if err := tmpl.ExecuteTemplate(&buf, "zonas.html", zdata); err != nil {
		t.Fatalf("execute zonas.html: %v", err)
	}

	zonas := buf.String()

	if !strings.Contains(zonas, "Dibujada") {
		t.Error("zonas.html should show drawn badge")
	}

	if !strings.Contains(zonas, "/admin/b2b#dibujar") {
		t.Error("zonas.html should send users to map drawing")
	}

	if !strings.Contains(zonas, `id="btn-draw-zone"`) {
		t.Error("zonas.html should include the draw-zone button")
	}

	for _, needle := range []string{
		`id="btn-draw-straight-zone"`,
		`/admin/b2b#dibujar-recto`,
		`Exportar Excel`,
		`Importar Excel`,
		`/admin/b2b/zones/`,
		`/export`,
		`/import`,
		`/embed/zona?t=`,
		`Enlace público`,
		`Editar dibujo`,
		`#dibujar-zona-`,
		`/admin/b2b/zone-groups`,
		`Grupo de zonas`,
		`/embed/grupo?t=`,
		`Norte unido`,
		`editGroup`,
		`id="edit-group-modal"`,
	} {
		if !strings.Contains(zonas, needle) {
			t.Errorf("zonas.html missing %q", needle)
		}
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
		"Cities":            []string{"Bogotá", "Chía", "Tunja"},
		"Departments":       Departments(),
		"DepartmentsData":   asJSON(Departments()),
		"DeptVal":           PlaceAll,
		"CityVal":           PlaceAll,
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
		`id="f-rating"`,
		`id="btn-map-full"`,
		`Pantalla completa`,
		`value="featured"`,
		`Destacado`,
		`PRICE_SMART`,
		`paintPriceSmart`,
		`PriceSmart Chía`,
		`key === "bogota"`,
		`key === "chia"`,
		`jobs-title`,
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
		`id="btn-reset-leads"`,
		`name="rating"`,
		`2 a 3`,
		`3 a 4`,
		`4 a 5`,
		`value="featured"`,
		`target="_blank"`,
		`id="bulk-form"`,
		`novalidate`,
		`trash-one`,
		`Confirmar papelera`,
		`Enviar a papelera`,
		`valueSel.disabled = isDelete`,
	} {
		if !strings.Contains(neg, needle) {
			t.Errorf("negocios.html missing %q", needle)
		}
	}

	if strings.Contains(neg, `confirm("¿Eliminar`) {
		t.Error("negocios.html must not block trash with window.confirm")
	}
}

func TestEmbedPopupUsesReadableInk(t *testing.T) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}

	data := map[string]any{
		"Token":             "z.1.1.abc",
		"Zone":              Zone{ID: 1, Name: "Norte", City: "Bogotá", Color: "#0891b2"},
		"ZoneData":          asJSON([]Zone{{ID: 1, Name: "Norte"}}),
		"GoogleMapsAPIKey":  "AIza-test",
		"AssetVersion":      "test",
		"Cities":            []string{"Bogotá"},
		"CityVal":           "Bogotá",
		"Advisors":          []Advisor{},
		"Zones":             []Zone{},
		"Categories":        []Category{},
		"AdvisorsData":      asJSON([]Advisor{}),
		"ZonesData":         asJSON([]Zone{}),
		"CategoriesData":    asJSON([]Category{}),
		"PriceSmartData":    asJSON([]any{}),
		"BogotaLocalidades": BogotaUrbanLocalidades(),
		"Group":             ZoneGroup{ID: 1, Name: "Norte unido"},
	}

	for _, name := range []string{"embed_zona.html", "embed_map.html", "embed_grupo.html"} {
		var buf strings.Builder
		if err := tmpl.ExecuteTemplate(&buf, name, data); err != nil {
			t.Fatalf("execute %s: %v", name, err)
		}

		html := buf.String()
		for _, needle := range []string{
			`.gm-style-iw`,
			`.pop h4 { margin: 0 0 .35rem; font-size: 1rem; font-weight: 700; color: #0f172a;`,
			`.pop .meta { font-size: .85rem; line-height: 1.6; color: #334155;`,
			`.pop .meta a { color: #1d4ed8;`,
		} {
			if !strings.Contains(html, needle) {
				t.Errorf("%s missing readable popup style %q", name, needle)
			}
		}

		if strings.Contains(html, `.pop h4 { margin: 0 0 .35rem; font-size: 1rem; font-weight: 700; color: var(--ink);`) {
			t.Errorf("%s popup title still inherits dark-scheme --ink", name)
		}
	}
}
