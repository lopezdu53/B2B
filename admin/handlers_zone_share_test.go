//nolint:testpackage // exercises unexported zone-share helpers
package admin

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
)

func TestZoneShareTokenRoundTrip(t *testing.T) {
	secret := []byte("test-secret-key-000000000000000000000000")

	tok := zoneShareToken(7, 3, secret)
	tid, zid, ok := parseZoneShareToken(tok, secret)
	if !ok || tid != 7 || zid != 3 {
		t.Fatalf("round trip: ok=%v tid=%d zid=%d tok=%q", ok, tid, zid, tok)
	}

	if _, _, ok := parseGroupShareToken(tok, secret); ok {
		t.Fatal("zone token must not parse as a group token")
	}

	if _, _, ok := parseZoneShareToken(embedToken(7, secret), secret); ok {
		t.Fatal("tenant embed token must not parse as a zone token")
	}

	if _, ok := parseEmbedToken(tok, secret); ok {
		t.Fatal("zone token must not parse as a tenant embed token")
	}

	if _, _, ok := parseZoneShareToken("z.7.3.deadbeef", secret); ok {
		t.Fatal("forged zone token accepted")
	}

	other := zoneShareToken(7, 3, []byte("a-completely-different-secret-key-000000"))
	if _, _, ok := parseZoneShareToken(other, secret); ok {
		t.Fatal("zone token from different secret accepted")
	}
}

func TestZoneFileSlugAndImportCSV(t *testing.T) {
	if got := zoneFileSlug("Norte de Chapinero"); got != "norte-de-chapinero" {
		t.Fatalf("slug: %q", got)
	}

	csvBody := "Key,Negocio\nChIJabc,Taquería\n,Otro Nombre\n"
	rows, err := parseBusinessImportCSV(strings.NewReader(csvBody))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(rows) != 2 || rows[0].Key != "ChIJabc" || rows[1].Title != "Otro Nombre" {
		t.Fatalf("rows: %+v", rows)
	}

	semi := "Key;Negocio\nxyz;Café\n"
	semiRows, err := parseBusinessImportCSV(strings.NewReader(semi))
	if err != nil || len(semiRows) != 1 || semiRows[0].Key != "xyz" {
		t.Fatalf("semicolon csv: %v %+v", err, semiRows)
	}

	keys, skipped := matchImportedBusinessKeys(rows, []MapBusiness{
		{Key: "ChIJabc", Title: "Taquería"},
		{Key: "other", Title: "Otro Nombre"},
		{Key: "skip-me", Title: "No está"},
	})
	if len(keys) != 2 || skipped != 0 {
		t.Fatalf("match keys=%v skipped=%d", keys, skipped)
	}
}

func TestRequestOriginAndPublicURL(t *testing.T) {
	t.Parallel()

	r, _ := http.NewRequest(http.MethodGet, "http://localhost/admin/b2b/zonas", nil)
	r.Host = "localhost:8080"
	r.Header.Set("X-Forwarded-Proto", "https")
	r.Header.Set("X-Forwarded-Host", "b2b.ventabot.cloud")
	r.TLS = &tls.ConnectionState{}

	if got := requestOrigin(r); got != "https://b2b.ventabot.cloud" {
		t.Fatalf("origin: %q", got)
	}

	got := zonePublicURL(r, 1, 2, []byte("secret"))
	if !strings.HasPrefix(got, "https://b2b.ventabot.cloud/embed/zona?t=z.1.2.") {
		t.Fatalf("public url: %q", got)
	}

	groupURL := groupPublicURL(r, 1, 9, []byte("secret"))
	if !strings.HasPrefix(groupURL, "https://b2b.ventabot.cloud/embed/grupo?t=g.1.9.") {
		t.Fatalf("group public url: %q", groupURL)
	}
}

func TestGroupShareTokenRoundTrip(t *testing.T) {
	secret := []byte("test-secret-key-000000000000000000000000")

	tok := groupShareToken(7, 4, secret)
	tid, gid, ok := parseGroupShareToken(tok, secret)
	if !ok || tid != 7 || gid != 4 {
		t.Fatalf("round trip: ok=%v tid=%d gid=%d tok=%q", ok, tid, gid, tok)
	}

	if !strings.HasPrefix(tok, "g.7.4.") {
		t.Fatalf("prefix: %q", tok)
	}

	if _, _, ok := parseGroupShareToken(zoneShareToken(7, 4, secret), secret); ok {
		t.Fatal("zone token must not parse as a group token")
	}

	if _, _, ok := parseZoneShareToken(tok, secret); ok {
		t.Fatal("group token must not parse as a zone token")
	}

	if _, ok := parseEmbedToken(tok, secret); ok {
		t.Fatal("group token must not parse as a tenant embed token")
	}

	if _, _, ok := parseGroupShareToken("g.7.4.deadbeef", secret); ok {
		t.Fatal("forged group token accepted")
	}

	other := groupShareToken(7, 4, []byte("a-completely-different-secret-key-000000"))
	if _, _, ok := parseGroupShareToken(other, secret); ok {
		t.Fatal("group token from different secret accepted")
	}
}

func TestParseZoneIDForm(t *testing.T) {
	r, err := http.NewRequest(http.MethodPost, "/", strings.NewReader("zone_id=3&zone_id=1&zone_id=1&zone_id=abc&zone_id=-2"))
	if err != nil {
		t.Fatal(err)
	}

	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}

	got := parseZoneIDForm(r)
	if len(got) != 2 || got[0] != 3 || got[1] != 1 {
		t.Fatalf("ids: %v", got)
	}
}
