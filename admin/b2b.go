package admin

import (
	"context"
	"encoding/json"
	"strings"
	"time"
)

// B2B CRM domain types and store contract.
//
// The scraper stores raw Google Maps businesses in the scrape_results table.
// The B2B layer adds a commercial overlay on top of those businesses: each one
// can be marked as a client / prospect / etc., and assigned to a sales advisor
// and a territorial zone.

// Business status values used by the B2B CRM overlay.
const (
	StatusProspect   = "prospect"    // negocio nuevo, candidato a cliente
	StatusInProgress = "in_progress" // en gestión comercial
	StatusClient     = "client"      // ya es cliente
	StatusFeatured   = "featured"    // destacado
	StatusDiscarded  = "discarded"   // descartado / no aplica
)

// ValidBusinessStatus reports whether s is a recognized CRM status.
func ValidBusinessStatus(s string) bool {
	switch s {
	case StatusProspect, StatusInProgress, StatusClient, StatusFeatured, StatusDiscarded:
		return true
	default:
		return false
	}
}

// Advisor is a commercial sales representative.
type Advisor struct {
	ID        int64
	Name      string
	Email     string
	Phone     string
	City      string
	Active    bool
	CreatedAt time.Time
}

// Zone is a territorial area (belongs to a city, optionally owned by an advisor).
type Zone struct {
	ID        int64
	Name      string
	City      string
	AdvisorID *int64
	Color     string
	Geometry  json.RawMessage `json:"Geometry,omitempty"`
	CreatedAt time.Time
}

// HasGeometry reports whether the zone has a drawn polygon on the map.
func (z *Zone) HasGeometry() bool {
	return z != nil && len(z.Geometry) > 0
}

// Category is a per-tenant business category (Restaurantes, Hoteles, …).
type Category struct {
	ID        int64
	Name      string
	Color     string
	Icon      string
	Count     int
	CreatedAt time.Time
}

// MapBusiness is a scraped business enriched with its CRM overlay, ready to be
// plotted on the map.
type MapBusiness struct {
	Key         string  `json:"key"`
	Title       string  `json:"title"`
	Category    string  `json:"category"`
	Address     string  `json:"address"`
	City        string  `json:"city"`
	Phone       string  `json:"phone"`
	Website     string  `json:"website"`
	MapsURL     string  `json:"maps_url"`
	Email       string  `json:"email"`
	Specialty   string  `json:"specialty"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	Rating      float64 `json:"rating"`
	ReviewCount int     `json:"review_count"`
	Status      string  `json:"status"`
	AdvisorID   *int64  `json:"advisor_id"`
	ZoneID      *int64  `json:"zone_id"`
	CategoryID  *int64  `json:"category_id"`
	Notes       string  `json:"notes"`
}

// AbsoluteHTTPURL prefixes https:// when a scraped site has no scheme.
func AbsoluteHTTPURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	lower := strings.ToLower(raw)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return raw
	}

	return "https://" + raw
}

// WebsiteURL is the clickable website for this business.
func (m MapBusiness) WebsiteURL() string {
	return AbsoluteHTTPURL(m.Website)
}

// EmailURL is a mailto link, or empty.
func (m MapBusiness) EmailURL() string {
	email := strings.TrimSpace(m.Email)
	if email == "" {
		return ""
	}

	return "mailto:" + email
}

// BusinessFilter narrows down the businesses returned for the map.
type BusinessFilter struct {
	City       string
	Status     string
	AdvisorID  *int64
	CategoryID *int64
	Search     string // free-text match on title / category / address
	Sort       string // "name" (default) | "category" | "added"
	Hidden     bool   // true = only trashed businesses (Papelera)
	Limit      int
	Offset     int
}

// B2BSummary holds aggregate counters for the dashboard header.
type B2BSummary struct {
	Total      int
	Clients    int
	Prospects  int
	InProgress int
	Discarded  int
	Featured   int
	Advisors   int
	Zones      int
}

// IB2BStore is the persistence contract for the B2B CRM layer. It is embedded
// into IStore so handlers can reach it through appState.Store.
type IB2BStore interface {
	// Businesses (scraped + per-tenant CRM overlay)
	ListBusinesses(ctx context.Context, tenantID int64, f BusinessFilter) ([]MapBusiness, error)
	CountBusinesses(ctx context.Context, tenantID int64, f BusinessFilter) (int, error)
	ListBusinessCities(ctx context.Context) ([]string, error)
	SetBusinessCRM(ctx context.Context, tenantID int64, key, status string, advisorID, zoneID *int64, notes, title string) error
	B2BSummary(ctx context.Context, tenantID int64, advisorID *int64) (*B2BSummary, error)
	RecordSearchJob(ctx context.Context, jobID, tenantID int64, rubro, specialty string) error
	IngestSearchJob(ctx context.Context, jobID int64) error
	IngestPendingSearchJobs(ctx context.Context) error
	HideAllVisibleBusinesses(ctx context.Context, tenantID int64) (int64, error)

	// Advisors
	ListAdvisors(ctx context.Context, tenantID int64) ([]Advisor, error)
	CreateAdvisor(ctx context.Context, tenantID int64, name, email, phone, city string) (*Advisor, error)
	UpdateAdvisor(ctx context.Context, tenantID, id int64, name, email, phone, city string) error
	DeleteAdvisor(ctx context.Context, tenantID, id int64) error

	// Zones
	ListZones(ctx context.Context, tenantID int64) ([]Zone, error)
	CreateZone(ctx context.Context, tenantID int64, name, city string, advisorID *int64, color string, geometry json.RawMessage) (*Zone, error)
	UpdateZone(ctx context.Context, tenantID, id int64, name, city string, advisorID *int64, color string, geometry json.RawMessage) error
	DeleteZone(ctx context.Context, tenantID, id int64) error

	// Categories
	ListCategories(ctx context.Context, tenantID int64) ([]Category, error)
	CreateCategory(ctx context.Context, tenantID int64, name, color, icon string) (*Category, error)
	UpdateCategory(ctx context.Context, tenantID, id int64, name, color, icon string) error
	DeleteCategory(ctx context.Context, tenantID, id int64) error
	ReassignCategory(ctx context.Context, tenantID, fromID, toID int64) error
	SetBusinessCategory(ctx context.Context, tenantID int64, key string, categoryID *int64) error

	// Bulk CRM operations on a set of business keys (each is an upsert)
	BulkSetStatus(ctx context.Context, tenantID int64, keys []string, status string) error
	BulkSetAdvisor(ctx context.Context, tenantID int64, keys []string, advisorID int64) error
	BulkSetZone(ctx context.Context, tenantID int64, keys []string, zoneID int64) error
	BulkSetCategory(ctx context.Context, tenantID int64, keys []string, categoryID int64) error
	BulkSetHidden(ctx context.Context, tenantID int64, keys []string, hidden bool) error

	// OwnedBusinessKeys returns the subset of keys assigned to advisorID (and not trashed).
	OwnedBusinessKeys(ctx context.Context, tenantID, advisorID int64, keys []string) ([]string, error)
}
