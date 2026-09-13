//nolint:testpackage // renders the embedded templates FS, which is unexported
package admin

import (
	"html/template"
	"io"
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

	advisors := []Advisor{{ID: advisorID, Name: "Ana", City: advisorCity, Email: "ana@example.com", Phone: "300"}}
	zones := []Zone{{ID: 1, Name: "Centro", City: advisorCity, AdvisorID: &advisorID, Color: "#2563eb"}}

	data := map[string]any{
		"CSRFToken":    "test-token",
		"Advisors":     advisors,
		"Zones":        zones,
		"AdvisorsData": asJSON(advisors),
		"ZonesData":    asJSON(zones),
		"Cities":       []string{"Bogotá", "Medellín"},
		"Summary":      &B2BSummary{Total: 10, Clients: 2, Prospects: 6, InProgress: 1, Discarded: 1, Advisors: 1, Zones: 1},
		"Success":      "",
		"Error":        "",
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
		"CityVal":          "",
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
		"Token":            "1.deadbeef",
		"Cities":           []string{"Bogotá", "Medellín"},
		"Advisors":         advisors,
		"Zones":            zones,
		"Categories":       []Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 3}},
		"AdvisorsData":     asJSON(advisors),
		"ZonesData":        asJSON(zones),
		"CategoriesData":   asJSON([]Category{{ID: 1, Name: "Restaurantes", Color: "#e11d48", Icon: "🍽️", Count: 3}}),
		"GoogleMapsAPIKey": "AIza-test",
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
	if err := tmpl.ExecuteTemplate(io.Discard, "categorias.html", catData); err != nil {
		t.Fatalf("execute categorias.html: %v", err)
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
