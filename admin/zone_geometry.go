package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	geoTypePolygon      = "Polygon"
	geoTypeMultiPolygon = "MultiPolygon"
	geoTypeFeature      = "Feature"
)

// NormalizeZoneGeometry validates a GeoJSON Polygon (or Feature wrapping one)
// and returns compact JSON suitable for storage.
func NormalizeZoneGeometry(raw string) (json.RawMessage, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return nil, fmt.Errorf("geometría inválida")
	}

	typ, _ := doc["type"].(string)
	if typ == geoTypeFeature {
		geom, _ := doc["geometry"].(map[string]any)
		if geom == nil {
			return nil, fmt.Errorf("el dibujo no tiene geometría")
		}

		doc = geom
		typ, _ = doc["type"].(string)
	}

	if typ != geoTypePolygon && typ != geoTypeMultiPolygon {
		return nil, fmt.Errorf("dibuja un polígono cerrado")
	}

	coords, ok := doc["coordinates"]
	if !ok {
		return nil, fmt.Errorf("el polígono no tiene coordenadas")
	}

	if typ == "Polygon" {
		if err := validatePolygonCoords(coords); err != nil {
			return nil, err
		}
	} else {
		polys, ok := coords.([]any)
		if !ok || len(polys) == 0 {
			return nil, fmt.Errorf("el polígono está vacío")
		}

		for _, poly := range polys {
			if err := validatePolygonCoords(poly); err != nil {
				return nil, err
			}
		}
	}

	out, err := json.Marshal(map[string]any{
		"type":        typ,
		"coordinates": coords,
	})
	if err != nil {
		return nil, fmt.Errorf("geometría inválida")
	}

	return out, nil
}

// NormalizeZoneGeometryJSON accepts a JSON value that is either a GeoJSON
// object or a string containing one.
func NormalizeZoneGeometryJSON(raw json.RawMessage) (json.RawMessage, error) {
	raw = json.RawMessage(bytes.TrimSpace(raw))
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, fmt.Errorf("geometría inválida")
		}

		return NormalizeZoneGeometry(s)
	}

	return NormalizeZoneGeometry(string(raw))
}

// PointInZoneGeometry reports whether lat/lng sits inside a GeoJSON Polygon or
// MultiPolygon (holes punch out, same rule as the map's pointInGeom).
func PointInZoneGeometry(lat, lng float64, raw json.RawMessage) bool {
	raw = json.RawMessage(bytes.TrimSpace(raw))
	if len(raw) == 0 || string(raw) == "null" {
		return false
	}

	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false
	}

	typ, _ := doc["type"].(string)
	if typ == geoTypeFeature {
		geom, _ := doc["geometry"].(map[string]any)
		if geom == nil {
			return false
		}

		doc = geom
		typ, _ = doc["type"].(string)
	}

	switch typ {
	case geoTypePolygon:
		return pointInPolygonCoords(lat, lng, doc["coordinates"])
	case geoTypeMultiPolygon:
		polys, ok := doc["coordinates"].([]any)
		if !ok {
			return false
		}

		for _, poly := range polys {
			if pointInPolygonCoords(lat, lng, poly) {
				return true
			}
		}
	}

	return false
}

func pointInPolygonCoords(lat, lng float64, coords any) bool {
	rings, ok := coords.([]any)
	if !ok || len(rings) == 0 {
		return false
	}

	outer, ok := ringLngLat(rings[0])
	if !ok || !pointInRing(lat, lng, outer) {
		return false
	}

	for i := 1; i < len(rings); i++ {
		hole, ok := ringLngLat(rings[i])
		if ok && pointInRing(lat, lng, hole) {
			return false
		}
	}

	return true
}

func ringLngLat(raw any) ([][2]float64, bool) {
	pts, ok := raw.([]any)
	if !ok || len(pts) < 3 {
		return nil, false
	}

	out := make([][2]float64, 0, len(pts))
	for _, p := range pts {
		pair, ok := p.([]any)
		if !ok || len(pair) < 2 {
			return nil, false
		}

		lng, ok1 := jsonFloat(pair[0])
		lat, ok2 := jsonFloat(pair[1])
		if !ok1 || !ok2 {
			return nil, false
		}

		out = append(out, [2]float64{lng, lat})
	}

	return out, true
}

func jsonFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func pointInRing(lat, lng float64, ring [][2]float64) bool {
	inside := false
	j := len(ring) - 1

	for i := 0; i < len(ring); i++ {
		xi, yi := ring[i][0], ring[i][1]
		xj, yj := ring[j][0], ring[j][1]
		denom := yj - yi
		if denom == 0 {
			denom = 1e-12
		}

		intersect := ((yi > lat) != (yj > lat)) && (lng < (xj-xi)*(lat-yi)/denom+xi)
		if intersect {
			inside = !inside
		}

		j = i
	}

	return inside
}

// BusinessInZone reports whether a lead belongs to the zone by CRM assignment
// or by sitting inside the drawn polygon.
func BusinessInZone(b *MapBusiness, z *Zone) bool {
	if b == nil || z == nil {
		return false
	}

	if z.ID != 0 && b.ZoneID != nil && *b.ZoneID == z.ID {
		return true
	}

	if b.Lat == 0 && b.Lng == 0 {
		return false
	}

	return PointInZoneGeometry(b.Lat, b.Lng, z.Geometry)
}

// FilterBusinessesInZone keeps the businesses that belong to z.
func FilterBusinessesInZone(list []MapBusiness, z *Zone) []MapBusiness {
	if z == nil {
		return nil
	}

	return FilterBusinessesInZones(list, []Zone{*z})
}

// FilterBusinessesInZones keeps businesses that belong to any of the zones
// (CRM assignment or polygon), de-duplicated by Key.
func FilterBusinessesInZones(list []MapBusiness, zones []Zone) []MapBusiness {
	if len(zones) == 0 {
		return nil
	}

	out := make([]MapBusiness, 0, len(list))
	seen := make(map[string]struct{}, len(list))

	for i := range list {
		key := strings.TrimSpace(list[i].Key)
		if key == "" {
			key = fmt.Sprintf("#%d", i)
		}

		if _, dup := seen[key]; dup {
			continue
		}

		for j := range zones {
			if BusinessInZone(&list[i], &zones[j]) {
				seen[key] = struct{}{}
				out = append(out, list[i])

				break
			}
		}
	}

	return out
}

func validatePolygonCoords(coords any) error {
	rings, ok := coords.([]any)
	if !ok || len(rings) == 0 {
		return fmt.Errorf("el polígono está vacío")
	}

	ring, ok := rings[0].([]any)
	if !ok || len(ring) < 4 {
		return fmt.Errorf("marca al menos 3 puntos y cierra el polígono")
	}

	return nil
}
