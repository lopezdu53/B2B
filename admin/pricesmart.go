package admin

// PriceSmartPOI is a warehouse club we always pin on the map.
type PriceSmartPOI struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	City    string  `json:"city"`
	Address string  `json:"address"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Phone   string  `json:"phone"`
	Website string  `json:"website"`
	MapsURL string  `json:"maps_url"`
}

const priceSmartSite = "https://www.pricesmart.com/en-co/clubs-and-locations"

// PriceSmartLocations is Bogotá (Salitre + Usaquén) and Chía.
// Coordinates match the OpenStreetMap footprints of each club.
func PriceSmartLocations() []PriceSmartPOI {
	return []PriceSmartPOI{
		{
			ID:      "ps-salitre",
			Name:    "PriceSmart Salitre",
			City:    "Bogotá",
			Address: "Av. Calle 26 # 71A-16, Salitre, Bogotá",
			Lat:     4.6657282,
			Lng:     -74.1099982,
			Phone:   "+57 601 7424114",
			Website: priceSmartSite,
			MapsURL: "https://www.google.com/maps/search/?api=1&query=PriceSmart+Salitre+Calle+26+71A+Bogota",
		},
		{
			ID:      "ps-usaquen",
			Name:    "PriceSmart Usaquén",
			City:    "Bogotá",
			Address: "Av. Calle 170 / Carrera 8 # 167D-88, Usaquén, Bogotá",
			Lat:     4.7464325,
			Lng:     -74.0245644,
			Phone:   "+57 601 7424114",
			Website: priceSmartSite,
			MapsURL: "https://www.google.com/maps/search/?api=1&query=PriceSmart+Usaquen+Calle+170+Carrera+8+Bogota",
		},
		{
			ID:      "ps-chia",
			Name:    "PriceSmart Chía",
			City:    "Chía",
			Address: "Vía Simandoy, Yerbabuena Bajo, Autopista Norte, Chía",
			Lat:     4.8873559,
			Lng:     -74.0110709,
			Phone:   "+57 601 7424114",
			Website: priceSmartSite,
			MapsURL: "https://www.google.com/maps/search/?api=1&query=PriceSmart+Chia+Yerbabuena",
		},
	}
}
