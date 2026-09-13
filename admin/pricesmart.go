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
func PriceSmartLocations() []PriceSmartPOI {
	return []PriceSmartPOI{
		{
			ID:      "ps-salitre",
			Name:    "PriceSmart Salitre",
			City:    "Bogotá",
			Address: "Av. Calle 80 # 69-70, Salitre, Bogotá",
			Lat:     4.6869,
			Lng:     -74.0853,
			Phone:   "+57 601 7424114",
			Website: priceSmartSite,
			MapsURL: "https://www.google.com/maps/search/?api=1&query=PriceSmart+Salitre+Calle+80+Bogota",
		},
		{
			ID:      "ps-usaquen",
			Name:    "PriceSmart Usaquén",
			City:    "Bogotá",
			Address: "Autopista Norte # 183-05, Usaquén, Bogotá",
			Lat:     4.7605,
			Lng:     -74.0462,
			Phone:   "+57 601 7424114",
			Website: priceSmartSite,
			MapsURL: "https://www.google.com/maps/search/?api=1&query=PriceSmart+Usaquen+Autopista+Norte+Bogota",
		},
		{
			ID:      "ps-chia",
			Name:    "PriceSmart Chía",
			City:    "Chía",
			Address: "Autopista Norte Km 10.3, Yerbabuena, Chía",
			Lat:     4.8368,
			Lng:     -74.0348,
			Phone:   "+57 601 7424114",
			Website: priceSmartSite,
			MapsURL: "https://www.google.com/maps/search/?api=1&query=PriceSmart+Chia+Autopista+Norte",
		},
	}
}
