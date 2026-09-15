//nolint:testpackage // reads the embedded static FS
package admin

import (
	"strings"
	"testing"
)

func TestB2BMapUsesLocatorPins(t *testing.T) {
	t.Parallel()

	raw, err := staticFS.ReadFile("static/b2b-map.js")
	if err != nil {
		t.Fatalf("read b2b-map.js: %v", err)
	}

	js := string(raw)
	for _, needle := range []string{
		"function pinDropHTML",
		"b2b-gpin-drop",
		"placeGooglePin",
		"leafletLocatorIcon",
		"OverlayView",
		`PS_PIN_COLOR`,
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("b2b-map.js missing %q", needle)
		}
	}

	for _, gone := range []string{
		"AdvancedMarkerElement",
		"DEMO_MAP_ID",
		"googleLocatorIcon",
		"googleSymbolPin",
		"MAPS_PIN_PATH",
		"google.maps.marker",
	} {
		if strings.Contains(js, gone) {
			t.Errorf("b2b-map.js should not use %q", gone)
		}
	}

	idx := strings.Index(js, "addMarker: function (biz, color, onClick)")
	if idx < 0 {
		t.Fatal("addMarker missing")
	}

	snippet := js[idx:]
	if len(snippet) > 400 {
		snippet = snippet[:400]
	}

	if strings.Contains(snippet, "circleMarker") || strings.Contains(snippet, "SymbolPath.CIRCLE") {
		t.Error("business addMarker should use locator pins, not circles")
	}

	if strings.Contains(js, "createGooglePSOverlay") {
		t.Error("PriceSmart should use the same locator pin as businesses")
	}
}

func TestB2BMapSupportsStraightDraw(t *testing.T) {
	t.Parallel()

	raw, err := staticFS.ReadFile("static/b2b-map.js")
	if err != nil {
		t.Fatalf("read b2b-map.js: %v", err)
	}

	js := string(raw)
	for _, needle := range []string{
		"drawOpts.mode",
		`mode === "straight"`,
		"startStreetDraw(engine, color, drawOpts.onVertex, onDone, drawOpts.mode)",
		"geomBounds: geomBounds",
	} {
		if !strings.Contains(js, needle) {
			t.Errorf("b2b-map.js missing %q", needle)
		}
	}
}
