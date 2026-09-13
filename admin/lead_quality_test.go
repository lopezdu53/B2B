package admin

import "testing"

func TestIsExcludedChain(t *testing.T) {
	t.Parallel()

	banned := []string{
		"Éxito Salitre",
		"Almacenes Éxito",
		"Olímpica Calle 80",
		"D1",
		"Tiendas D1 Suba",
		"Ara",
		"Supermercado Ara",
		"Carulla FreshMarket",
		"Oxxo Chapinero",
		"Falabella Centro Mayor",
		"Justo & Bueno",
		"Justo y Bueno Kennedy",
		"Ísimo",
		"Homecenter Calle 80",
		"Easy Bogotá",
	}
	for _, name := range banned {
		if !IsExcludedChain(name) {
			t.Errorf("expected excluded: %q", name)
		}
	}

	keep := []string{
		"Restaurante El Araucano",
		"Pizzería Don Julio",
		"Hotel Ibis",
		"PriceSmart Calle 80",
		"Fruver de la 85",
	}
	for _, name := range keep {
		if IsExcludedChain(name) {
			t.Errorf("should keep: %q", name)
		}
	}
}

func TestQualifiesAsLead(t *testing.T) {
	t.Parallel()

	if QualifiesAsLead("Restaurante Andino", 50) {
		t.Fatal("50 reviews is not enough")
	}

	if !QualifiesAsLead("Restaurante Andino", 51) {
		t.Fatal("51 reviews should qualify")
	}

	if QualifiesAsLead("Éxito", 900) {
		t.Fatal("Éxito should never qualify")
	}
}
