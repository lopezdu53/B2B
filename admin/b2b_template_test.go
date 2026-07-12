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
		"AdvisorsData": advisors,
		"ZonesData":    zones,
		"Cities":       []string{"Bogotá", "Medellín"},
		"Summary":      &B2BSummary{Total: 10, Clients: 2, Prospects: 6, InProgress: 1, Discarded: 1, Advisors: 1, Zones: 1},
		"Success":      "",
		"Error":        "",
		// list pages
		"Rows": []businessRow{
			{MapBusiness: MapBusiness{Title: "Rest", City: "Bogotá", Status: "client"}, StatusLabel: "Cliente", StatusClass: "client", AdvisorName: "Ana"},
		},
		"Count":   1,
		"QVal":    "",
		"CityVal": "",
		"StatVal": "",
		"AdvVal":  "",
	}

	for _, name := range []string{"b2b.html", "negocios.html", "asesores.html"} {
		if err := tmpl.ExecuteTemplate(io.Discard, name, data); err != nil {
			t.Fatalf("execute %s: %v", name, err)
		}
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
}
