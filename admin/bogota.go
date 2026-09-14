package admin

import (
	"encoding/json"
	"math"
	"strings"
	"sync"
)

const kmPerDegreeLat = 111.32

const sumapazCodigo = "20"

// BogotaLocalidad is a Bogotá locality plus its neighborhood names.
type BogotaLocalidad struct {
	Nombre  string   `json:"nombre"`
	Codigo  string   `json:"codigo"`
	Color   string   `json:"color"`
	Barrios []string `json:"barrios"`
}

// BogotaIndex is the compact locality/barrio directory used by the search form.
type BogotaIndex struct {
	Localidades []BogotaLocalidad `json:"localidades"`
}

var (
	bogotaIndexOnce sync.Once
	bogotaIndex     BogotaIndex
	bogotaIndexErr  error
)

func loadBogotaIndex() {
	bogotaIndexOnce.Do(func() {
		raw, err := staticFS.ReadFile("static/geo/bogota-index.json")
		if err != nil {
			bogotaIndexErr = err
			return
		}

		bogotaIndexErr = json.Unmarshal(raw, &bogotaIndex)
	})
}

// LoadBogotaIndex returns the official Bogotá locality/barrio index.
func LoadBogotaIndex() (BogotaIndex, error) {
	loadBogotaIndex()

	return bogotaIndex, bogotaIndexErr
}

// BogotaUrbanLocalidades returns urban localities for dropdowns (excludes Sumapaz).
func BogotaUrbanLocalidades() []BogotaLocalidad {
	idx, err := LoadBogotaIndex()
	if err != nil {
		return nil
	}

	out := make([]BogotaLocalidad, 0, len(idx.Localidades))
	for _, loc := range idx.Localidades {
		if loc.Codigo == sumapazCodigo {
			continue
		}

		out = append(out, loc)
	}

	return out
}

// LocalidadHint is the map viewport used when scraping a Bogotá localidad.
type LocalidadHint struct {
	Nombre string
	Lat    float64
	Lon    float64
	Zoom   int
}

type localidadFeatureCollection struct {
	Features []localidadFeature `json:"features"`
}

type localidadFeature struct {
	Properties struct {
		Nombre string `json:"nombre"`
	} `json:"properties"`
	Geometry struct {
		Type        string          `json:"type"`
		Coordinates json.RawMessage `json:"coordinates"`
	} `json:"geometry"`
}

var (
	localidadHintsOnce sync.Once
	localidadHints     map[string]LocalidadHint
	localidadHintsErr  error
)

func loadLocalidadHints() {
	localidadHintsOnce.Do(func() {
		raw, err := staticFS.ReadFile("static/geo/bogota-localidades.json")
		if err != nil {
			localidadHintsErr = err
			return
		}

		var fc localidadFeatureCollection
		if err := json.Unmarshal(raw, &fc); err != nil {
			localidadHintsErr = err
			return
		}

		localidadHints = make(map[string]LocalidadHint, len(fc.Features))

		for i := range fc.Features {
			hint, ok := hintFromLocalidadFeature(fc.Features[i])
			if !ok {
				continue
			}

			localidadHints[cityKey(hint.Nombre)] = hint
		}
	})
}

func hintFromLocalidadFeature(f localidadFeature) (LocalidadHint, bool) {
	name := strings.TrimSpace(f.Properties.Nombre)
	if name == "" {
		return LocalidadHint{}, false
	}

	ring := firstPolygonRing(f.Geometry.Type, f.Geometry.Coordinates)
	lat, lon, spanKm, ok := ringCentroidAndSpan(ring)

	if !ok {
		return LocalidadHint{}, false
	}

	return LocalidadHint{
		Nombre: name,
		Lat:    lat,
		Lon:    lon,
		Zoom:   zoomFromSpanKm(spanKm),
	}, true
}

func firstPolygonRing(kind string, raw json.RawMessage) [][]float64 {
	switch kind {
	case "MultiPolygon":
		var multi [][][][]float64
		if json.Unmarshal(raw, &multi) != nil || len(multi) == 0 || len(multi[0]) == 0 {
			return nil
		}

		return multi[0][0]
	default:
		var poly [][][]float64
		if json.Unmarshal(raw, &poly) != nil || len(poly) == 0 {
			return nil
		}

		return poly[0]
	}
}

func ringCentroidAndSpan(ring [][]float64) (lat, lon, spanKm float64, ok bool) {
	if len(ring) < 3 {
		return 0, 0, 0, false
	}

	var sumLat, sumLon float64

	minLat, maxLat := 90.0, -90.0
	minLon, maxLon := 180.0, -180.0
	n := 0

	for _, pt := range ring {
		if len(pt) < 2 {
			continue
		}

		ptLon, ptLat := pt[0], pt[1]
		sumLat += ptLat
		sumLon += ptLon

		if ptLat < minLat {
			minLat = ptLat
		}

		if ptLat > maxLat {
			maxLat = ptLat
		}

		if ptLon < minLon {
			minLon = ptLon
		}

		if ptLon > maxLon {
			maxLon = ptLon
		}

		n++
	}

	if n == 0 {
		return 0, 0, 0, false
	}

	lat = sumLat / float64(n)
	lon = sumLon / float64(n)
	spanLat := (maxLat - minLat) * kmPerDegreeLat
	cosLat := math.Cos(lat * math.Pi / 180)
	spanLon := (maxLon - minLon) * kmPerDegreeLat * math.Abs(cosLat)
	spanKm = spanLat

	if spanLon > spanKm {
		spanKm = spanLon
	}

	return lat, lon, spanKm, true
}

func zoomFromSpanKm(span float64) int {
	switch {
	case span >= 20:
		return 12
	case span >= 10:
		return 13
	case span >= 5:
		return 14
	default:
		return 15
	}
}

// LocalidadSearchHint returns the scrape viewport for a Bogotá localidad.
func LocalidadSearchHint(nombre string) (LocalidadHint, bool) {
	loadLocalidadHints()

	if localidadHintsErr != nil || localidadHints == nil {
		return LocalidadHint{}, false
	}

	hint, ok := localidadHints[cityKey(nombre)]

	return hint, ok
}
