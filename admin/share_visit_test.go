package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCountryFlag(t *testing.T) {
	if got := CountryFlag("co"); got != "🇨🇴" {
		t.Fatalf("CO flag: got %q", got)
	}

	if got := CountryFlag(""); got != "🌐" {
		t.Fatalf("empty iso: got %q", got)
	}
}

func TestFormatVisitClockBogota(t *testing.T) {
	at := time.Date(2026, 9, 19, 16, 5, 0, 0, time.UTC)
	got := FormatVisitClock(at, "America/Bogota")
	if !strings.Contains(got, "sábado") || !strings.Contains(got, "19 de septiembre de 2026") || !strings.Contains(got, "11:05") {
		t.Fatalf("clock = %q", got)
	}
}

func TestClientIPFromRequest(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/embed/zona", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "190.25.1.10, 10.0.0.2")

	if got := clientIPFromRequest(r); got != "190.25.1.10" {
		t.Fatalf("ip = %q", got)
	}
}

func TestLookupIPPlacePrivate(t *testing.T) {
	place := LookupIPPlace(context.Background(), "10.0.0.8", "CO")
	if place.City != "Red local" {
		t.Fatalf("private place = %+v", place)
	}
}

func TestParseIPWhoBody(t *testing.T) {
	body := []byte(`{"success":true,"country":"Colombia","country_code":"CO","city":"Bogotá","timezone":{"id":"America/Bogota"}}`)
	place := parseIPWhoBody(body)
	if place.Country != "Colombia" || place.CountryISO != "CO" || place.City != "Bogotá" || place.Timezone != "America/Bogota" {
		t.Fatalf("place = %+v", place)
	}
}

func TestLookupIPPlaceHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"success":true,"country":"Colombia","country_code":"CO","city":"Chía","timezone":{"id":"America/Bogota"}}`))
	}))
	t.Cleanup(srv.Close)

	prevURL, prevClient := ipWhoURL, ipWhoClient
	ipWhoURL = func(string) string { return srv.URL }
	ipWhoClient = srv.Client()
	t.Cleanup(func() {
		ipWhoURL = prevURL
		ipWhoClient = prevClient
	})

	place := LookupIPPlace(context.Background(), "190.25.1.10", "")
	if place.City != "Chía" || place.CountryISO != "CO" {
		t.Fatalf("lookup = %+v", place)
	}
}

func TestShareVisitLabels(t *testing.T) {
	v := ShareVisit{
		Kind:       ShareKindGroup,
		TargetName: "Norte",
		CountryISO: "CO",
		City:       "Tunja",
		Timezone:   "America/Bogota",
		VisitedAt:  time.Date(2026, 9, 19, 16, 5, 0, 0, time.UTC),
	}
	if v.KindLabel() != "Grupo" {
		t.Fatalf("kind = %q", v.KindLabel())
	}
	if v.FlagURL() != "https://flagcdn.com/24x18/co.png" {
		t.Fatalf("flag url = %q", v.FlagURL())
	}
	if v.PlaceLabel() != "Tunja" {
		t.Fatalf("place = %q", v.PlaceLabel())
	}
}

func TestIsPrivateIPString(t *testing.T) {
	if !isPrivateIPString("127.0.0.1") || !isPrivateIPString("192.168.1.5") || isPrivateIPString("8.8.8.8") {
		t.Fatal("private IP classification")
	}
}
