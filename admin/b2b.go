package admin

import (
	"context"
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
	StatusDiscarded  = "discarded"   // descartado / no aplica
)

// ValidBusinessStatus reports whether s is a recognized CRM status.
func ValidBusinessStatus(s string) bool {
	switch s {
	case StatusProspect, StatusInProgress, StatusClient, StatusDiscarded:
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
	CreatedAt time.Time
}

// MapBusiness is a scraped business enriched with its CRM overlay, ready to be
// plotted on the map.
type MapBusiness struct {
	Key       string  `json:"key"`
	Title     string  `json:"title"`
	Category  string  `json:"category"`
	Address   string  `json:"address"`
	City      string  `json:"city"`
	Phone     string  `json:"phone"`
	Website   string  `json:"website"`
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	Status    string  `json:"status"`
	AdvisorID *int64  `json:"advisor_id"`
	ZoneID    *int64  `json:"zone_id"`
	Notes     string  `json:"notes"`
}

// BusinessFilter narrows down the businesses returned for the map.
type BusinessFilter struct {
	City      string
	Status    string
	AdvisorID *int64
	Search    string // free-text match on title / category / address
	Limit     int
}

// B2BSummary holds aggregate counters for the dashboard header.
type B2BSummary struct {
	Total      int
	Clients    int
	Prospects  int
	InProgress int
	Discarded  int
	Advisors   int
	Zones      int
}

// IB2BStore is the persistence contract for the B2B CRM layer. It is embedded
// into IStore so handlers can reach it through appState.Store.
type IB2BStore interface {
	// Businesses (scraped + CRM overlay)
	ListBusinesses(ctx context.Context, f BusinessFilter) ([]MapBusiness, error)
	ListBusinessCities(ctx context.Context) ([]string, error)
	SetBusinessCRM(ctx context.Context, key, status string, advisorID, zoneID *int64, notes, title string) error
	B2BSummary(ctx context.Context) (*B2BSummary, error)

	// Advisors
	ListAdvisors(ctx context.Context) ([]Advisor, error)
	CreateAdvisor(ctx context.Context, name, email, phone, city string) (*Advisor, error)
	DeleteAdvisor(ctx context.Context, id int64) error

	// Zones
	ListZones(ctx context.Context) ([]Zone, error)
	CreateZone(ctx context.Context, name, city string, advisorID *int64, color string) (*Zone, error)
	DeleteZone(ctx context.Context, id int64) error
}
