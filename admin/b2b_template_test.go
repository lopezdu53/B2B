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

	data := map[string]any{
		"CSRFToken": "test-token",
		"Advisors": []Advisor{
			{ID: advisorID, Name: "Ana", City: advisorCity, Email: "ana@example.com", Phone: "300"},
		},
		"Zones": []Zone{
			{ID: 1, Name: "Centro", City: advisorCity, AdvisorID: &advisorID, Color: "#2563eb"},
		},
		"Cities":  []string{"Bogotá", "Medellín"},
		"Summary": &B2BSummary{Total: 10, Clients: 2, Prospects: 6, InProgress: 1, Discarded: 1, Advisors: 1, Zones: 1},
		"Success": "",
		"Error":   "",
	}

	if err := tmpl.ExecuteTemplate(io.Discard, "b2b.html", data); err != nil {
		t.Fatalf("execute b2b.html: %v", err)
	}
}
