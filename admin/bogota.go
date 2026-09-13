package admin

import (
	"encoding/json"
	"sync"
)

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
