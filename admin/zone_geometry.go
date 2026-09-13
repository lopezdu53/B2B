package admin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
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
	if typ == "Feature" {
		geom, _ := doc["geometry"].(map[string]any)
		if geom == nil {
			return nil, fmt.Errorf("el dibujo no tiene geometría")
		}

		doc = geom
		typ, _ = doc["type"].(string)
	}

	if typ != "Polygon" && typ != "MultiPolygon" {
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
