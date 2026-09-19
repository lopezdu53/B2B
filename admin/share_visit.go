package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"
	"unicode"

	_ "time/tzdata"
)

const (
	// ShareKindZone is a visit to a single-zone public map.
	ShareKindZone = "zone"
	// ShareKindGroup is a visit to a zone-group public map.
	ShareKindGroup  = "group"
	shareVisitUAMax = 500
	shareVisitIPMax = 64
)

// ShareVisit is one open of a public zone or zone-group map link.
type ShareVisit struct {
	ID         int64
	Kind       string
	TargetID   int64
	TargetName string
	IP         string
	Country    string
	CountryISO string
	City       string
	Timezone   string
	UserAgent  string
	VisitedAt  time.Time
}

// KindLabel is the Spanish map type shown on the Visitas page.
func (v ShareVisit) KindLabel() string {
	switch v.Kind {
	case ShareKindGroup:
		return "Grupo"
	default:
		return "Zona"
	}
}

// FlagEmoji is the regional-indicator flag for CountryISO, or a globe.
func (v ShareVisit) FlagEmoji() string {
	return CountryFlag(v.CountryISO)
}

// FlagURL is a PNG flag from flagcdn, or empty when ISO is unknown.
func (v ShareVisit) FlagURL() string {
	iso := strings.ToLower(strings.TrimSpace(v.CountryISO))
	if len(iso) != 2 {
		return ""
	}

	return "https://flagcdn.com/24x18/" + iso + ".png"
}

// PlaceLabel is city, else country, else a local-network hint.
func (v ShareVisit) PlaceLabel() string {
	if city := strings.TrimSpace(v.City); city != "" {
		return city
	}

	if country := strings.TrimSpace(v.Country); country != "" {
		return country
	}

	return "Ubicación desconocida"
}

// ClockLabel is weekday, date and time in the visitor timezone.
func (v ShareVisit) ClockLabel() string {
	return FormatVisitClock(v.VisitedAt, v.Timezone)
}

// IPPlace is the geolocation result for a client IP.
type IPPlace struct {
	Country    string
	CountryISO string
	City       string
	Timezone   string
}

func (p IPPlace) empty() bool {
	return strings.TrimSpace(p.Country) == "" &&
		strings.TrimSpace(p.CountryISO) == "" &&
		strings.TrimSpace(p.City) == ""
}

var (
	ipWhoURL = func(ip string) string {
		return "https://ipwho.is/" + ip
	}
	ipWhoClient  = &http.Client{Timeout: 2500 * time.Millisecond}
	ipPlaceCache sync.Map
)

type ipWhoResp struct {
	Success     bool   `json:"success"`
	Country     string `json:"country"`
	CountryCode string `json:"country_code"`
	City        string `json:"city"`
	Timezone    any    `json:"timezone"`
}

// CountryFlag returns a flag emoji for a two-letter ISO country code.
func CountryFlag(iso string) string {
	iso = strings.ToUpper(strings.TrimSpace(iso))
	if len(iso) != 2 {
		return "🌐"
	}

	a, b := rune(iso[0]), rune(iso[1])
	if a < 'A' || a > 'Z' || b < 'A' || b > 'Z' {
		return "🌐"
	}

	return string([]rune{0x1F1E6 + (a - 'A'), 0x1F1E6 + (b - 'A')})
}

// FormatVisitClock formats t in tz using Spanish weekday and month names.
func FormatVisitClock(t time.Time, tz string) string {
	if t.IsZero() {
		t = time.Now()
	}

	loc := time.Local
	if name := strings.TrimSpace(tz); name != "" {
		if loaded, err := time.LoadLocation(name); err == nil {
			loc = loaded
		}
	}

	local := t.In(loc)
	weekday := []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	month := []string{
		"", "enero", "febrero", "marzo", "abril", "mayo", "junio",
		"julio", "agosto", "septiembre", "octubre", "noviembre", "diciembre",
	}

	return fmt.Sprintf("%s, %d de %s de %d · %02d:%02d",
		weekday[local.Weekday()],
		local.Day(),
		month[local.Month()],
		local.Year(),
		local.Hour(),
		local.Minute(),
	)
}

// clientIPFromRequest prefers proxy headers used by EasyPanel/Cloudflare.
func clientIPFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}

	for _, key := range []string{"CF-Connecting-IP", "X-Real-IP", "X-Forwarded-For"} {
		if ip := firstPublicIP(r.Header.Get(key)); ip != "" {
			return ip
		}
	}

	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}

	return host
}

func firstPublicIP(raw string) string {
	fallback := ""

	for _, part := range strings.Split(raw, ",") {
		ip := strings.TrimSpace(part)
		ip = strings.Trim(ip, "[]")
		if ip == "" {
			continue
		}

		parsed, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}

		if !isPrivateAddr(parsed) {
			return parsed.String()
		}

		if fallback == "" {
			fallback = parsed.String()
		}
	}

	return fallback
}

func isPrivateAddr(ip netip.Addr) bool {
	return !ip.IsValid() || ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() || ip.IsMulticast()
}

func isPrivateIPString(ip string) bool {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return true
	}

	parsed, err := netip.ParseAddr(ip)
	if err != nil {
		return true
	}

	return isPrivateAddr(parsed)
}

// LookupIPPlace resolves city/country/timezone for ip. Private addresses
// are labeled "Red local" and never sent to the geo service.
func LookupIPPlace(ctx context.Context, ip string, countryHint string) IPPlace {
	ip = strings.TrimSpace(ip)
	if isPrivateIPString(ip) {
		return IPPlace{Country: "Red local", City: "Red local"}
	}

	if cached, ok := ipPlaceCache.Load(ip); ok {
		if place, ok := cached.(IPPlace); ok {
			return place
		}
	}

	place := lookupIPWho(ctx, ip)
	if place.empty() {
		place = placeFromCountryHint(countryHint)
	}

	if !place.empty() {
		ipPlaceCache.Store(ip, place)
	}

	return place
}

func placeFromCountryHint(hint string) IPPlace {
	iso := strings.ToUpper(strings.TrimSpace(hint))
	if len(iso) != 2 || iso == "XX" || iso == "T1" {
		return IPPlace{}
	}

	return IPPlace{CountryISO: iso, Country: iso}
}

func lookupIPWho(ctx context.Context, ip string) IPPlace {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ipWhoURL(ip), nil)
	if err != nil {
		return IPPlace{}
	}

	req.Header.Set("Accept", "application/json")

	resp, err := ipWhoClient.Do(req)
	if err != nil {
		return IPPlace{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return IPPlace{}
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return IPPlace{}
	}

	return parseIPWhoBody(body)
}

func parseIPWhoBody(body []byte) IPPlace {
	var raw ipWhoResp
	if err := json.Unmarshal(body, &raw); err != nil || !raw.Success {
		return IPPlace{}
	}

	return IPPlace{
		Country:    strings.TrimSpace(raw.Country),
		CountryISO: strings.ToUpper(strings.TrimSpace(raw.CountryCode)),
		City:       strings.TrimSpace(raw.City),
		Timezone:   timezoneIDFromJSON(raw.Timezone),
	}
}

func timezoneIDFromJSON(v any) string {
	switch tz := v.(type) {
	case string:
		return strings.TrimSpace(tz)
	case map[string]any:
		for _, key := range []string{"id", "timezone", "name"} {
			if s, ok := tz[key].(string); ok && strings.TrimSpace(s) != "" {
				return strings.TrimSpace(s)
			}
		}
	}

	return ""
}

func clipRunes(s string, max int) string {
	if max <= 0 || s == "" {
		return s
	}

	n := 0
	for i := range s {
		if n == max {
			return s[:i]
		}
		n++
	}

	return s
}

func sanitizeVisitText(s string, max int) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) && r != '\t' {
			return -1
		}

		return r
	}, s)

	return clipRunes(strings.TrimSpace(s), max)
}

func newShareVisit(kind string, targetID int64, targetName, ip, ua string, place IPPlace, at time.Time) ShareVisit {
	if at.IsZero() {
		at = time.Now()
	}

	return ShareVisit{
		Kind:       kind,
		TargetID:   targetID,
		TargetName: sanitizeVisitText(targetName, 200),
		IP:         sanitizeVisitText(ip, shareVisitIPMax),
		Country:    sanitizeVisitText(place.Country, 80),
		CountryISO: sanitizeVisitText(place.CountryISO, 8),
		City:       sanitizeVisitText(place.City, 80),
		Timezone:   sanitizeVisitText(place.Timezone, 80),
		UserAgent:  sanitizeVisitText(ua, shareVisitUAMax),
		VisitedAt:  at,
	}
}
