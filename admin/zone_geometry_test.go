//nolint:testpackage // exercises unexported geometry helpers in this package
package admin

import (
	"encoding/json"
	"testing"
)

func TestNormalizeZoneGeometry(t *testing.T) {
	t.Parallel()

	validPoly := `{"type":"Polygon","coordinates":[[[-74.1,4.6],[-74.0,4.6],[-74.0,4.7],[-74.1,4.6]]]}`

	got, err := NormalizeZoneGeometry(validPoly)
	if err != nil {
		t.Fatalf("valid polygon: %v", err)
	}

	if !json.Valid(got) {
		t.Fatalf("expected compact JSON, got %s", got)
	}

	feat := `{"type":"Feature","geometry":` + validPoly + `,"properties":{}}`
	if _, err := NormalizeZoneGeometry(feat); err != nil {
		t.Fatalf("feature wrapper: %v", err)
	}

	asJSON, err := NormalizeZoneGeometryJSON(json.RawMessage(validPoly))
	if err != nil || len(asJSON) == 0 {
		t.Fatalf("raw object: %v %s", err, asJSON)
	}

	quoted, _ := json.Marshal(validPoly)
	if _, err := NormalizeZoneGeometryJSON(quoted); err != nil {
		t.Fatalf("quoted string: %v", err)
	}

	if geom, err := NormalizeZoneGeometry(""); err != nil || geom != nil {
		t.Fatalf("empty should be nil, got %s %v", geom, err)
	}

	if _, err := NormalizeZoneGeometry(`{"type":"Point","coordinates":[-74,4]}`); err == nil {
		t.Fatal("expected error for Point")
	}

	if _, err := NormalizeZoneGeometry(`{"type":"Polygon","coordinates":[[[-74,4],[-74,5]]]}`); err == nil {
		t.Fatal("expected error for short ring")
	}
}

func TestZoneHasGeometry(t *testing.T) {
	t.Parallel()

	var empty Zone

	if empty.HasGeometry() {
		t.Fatal("empty zone should not report geometry")
	}

	z := Zone{Geometry: json.RawMessage(`{"type":"Polygon","coordinates":[]}`)}
	if !z.HasGeometry() {
		t.Fatal("zone with raw geometry should report HasGeometry")
	}
}

func TestPointInZoneGeometry(t *testing.T) {
	t.Parallel()

	poly := json.RawMessage(`{"type":"Polygon","coordinates":[[[-74.12,4.60],[-74.08,4.60],[-74.08,4.65],[-74.12,4.65],[-74.12,4.60]]]}`)
	hole := json.RawMessage(`{"type":"Polygon","coordinates":[[[-74.12,4.60],[-74.08,4.60],[-74.08,4.65],[-74.12,4.65],[-74.12,4.60]],[[-74.11,4.61],[-74.09,4.61],[-74.09,4.63],[-74.11,4.63],[-74.11,4.61]]]}`)

	if !PointInZoneGeometry(4.62, -74.10, poly) {
		t.Fatal("interior point should be inside")
	}

	if PointInZoneGeometry(4.70, -74.10, poly) {
		t.Fatal("exterior point should be outside")
	}

	if PointInZoneGeometry(4.62, -74.10, hole) {
		t.Fatal("point in hole should be outside")
	}

	zid := int64(9)
	inside := MapBusiness{Key: "in", Lat: 4.62, Lng: -74.10}
	assigned := MapBusiness{Key: "as", ZoneID: &zid, Lat: 0, Lng: 0}
	other := MapBusiness{Key: "out", Lat: 5, Lng: -74}

	z := Zone{ID: zid, Geometry: poly}
	if !BusinessInZone(&inside, &z) || !BusinessInZone(&assigned, &z) || BusinessInZone(&other, &z) {
		t.Fatal("BusinessInZone membership mismatch")
	}

	got := FilterBusinessesInZone([]MapBusiness{inside, assigned, other}, &z)
	if len(got) != 2 {
		t.Fatalf("filter: want 2 got %d", len(got))
	}
}
