package postgres

import (
	"strings"
	"testing"

	"github.com/gosom/google-maps-scraper/admin"
)

func TestCityMatchSQLAcceptsBogotaLocalidades(t *testing.T) {
	t.Parallel()

	sql := cityMatchSQL()
	if sql == "" {
		t.Fatal("empty city match SQL")
	}

	for _, needle := range []string{
		"usaquen",
		"chapinero",
		"barrios unidos",
		"kennedy",
		"chia",
		"soacha",
		"bogota",
	} {
		if !strings.Contains(sql, "'"+needle+"'") {
			t.Errorf("city match SQL missing %q", needle)
		}
	}

	if !strings.Contains(listBusinessesQuery, "complete_address") {
		t.Error("list query should read complete_address")
	}

	if !strings.Contains(listBusinessesQuery, "->>'state'") {
		t.Error("list query should expose state for Bogotá matching")
	}

	if !strings.Contains(countBusinessesQuery, "->>'borough'") {
		t.Error("count query should expose borough for Bogotá matching")
	}
}

func TestBogotaAliasKeysCoverUrbanLocalidades(t *testing.T) {
	t.Parallel()

	keys := map[string]struct{}{}
	for _, k := range admin.BogotaCityAliasKeys() {
		keys[k] = struct{}{}
	}

	for _, loc := range admin.BogotaUrbanLocalidades() {
		// cityKey is unexported; CanonicalCity maps each localidad to Bogotá.
		if admin.CanonicalCity(loc.Nombre) != admin.DefaultCity {
			t.Errorf("localidad %q should canonicalize to Bogotá", loc.Nombre)
		}
	}

	if _, ok := keys["usaquen"]; !ok {
		t.Fatal("missing usaquen alias")
	}
}

func TestScrapeQueriesTolerateNonArrayResults(t *testing.T) {
	t.Parallel()

	for _, q := range []string{listBusinessesQuery, countBusinessesQuery, businessTotalQuery} {
		if !strings.Contains(q, scrapeResultsArraySQL) {
			t.Errorf("query missing non-array guard: %s", q[:80])
		}
		if strings.Contains(q, "jsonb_array_elements(sr.results)") {
			t.Error("raw jsonb_array_elements(sr.results) still present")
		}
	}

	if !strings.Contains(adminLeadQualitySQL, "~ '^[0-9]+$'") {
		t.Error("review_count cast should reject non-integers")
	}

	if !strings.Contains(listBusinessesQuery, "rating >= $11") {
		t.Error("list query should filter star rating")
	}

	if !strings.Contains(countBusinessesQuery, "rating >= $8") {
		t.Error("count query should filter star rating")
	}
}
