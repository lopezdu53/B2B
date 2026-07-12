package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gosom/google-maps-scraper/log"
)

// statusMeta maps a CRM status to its Spanish label and CSS class.
func statusMeta(status string) (label, class string) {
	switch status {
	case StatusClient:
		return "Cliente", "client"
	case StatusInProgress:
		return "En gestión", "in_progress"
	case StatusDiscarded:
		return "Descartado", "discarded"
	default:
		return "Prospecto", "prospect"
	}
}

// b2bRedirectBack redirects to the page the request came from when it is an
// admin page, otherwise to fallback. Used after create/delete so the user
// returns to whichever management page they were on.
func b2bRedirectBack(w http.ResponseWriter, r *http.Request, fallback, key, msg string) {
	dest := fallback
	if ref := r.Referer(); ref != "" && strings.Contains(ref, "/admin/") {
		dest = ref
	}

	sep := "?"
	if strings.Contains(dest, "?") {
		sep = "&"
	}

	http.Redirect(w, r, dest+sep+key+"="+msg, http.StatusSeeOther)
}

// businessRow is a scraped business enriched with resolved advisor/zone names
// for the list view.
type businessRow struct {
	MapBusiness

	StatusLabel string
	StatusClass string
	AdvisorName string
	ZoneName    string
}

// NegociosPageHandler renders the searchable list of businesses.
func NegociosPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()

		f := BusinessFilter{
			City:   q.Get("city"),
			Status: q.Get("status"),
			Search: strings.TrimSpace(q.Get("q")),
		}

		if a := strings.TrimSpace(q.Get("advisor")); a != "" {
			if id, err := strconv.ParseInt(a, 10, 64); err == nil {
				f.AdvisorID = &id
			}
		}

		tid, _ := effectiveTenant(appState, r)

		advisors, _ := appState.Store.ListAdvisors(ctx, tid)
		zones, _ := appState.Store.ListZones(ctx, tid)
		cities, _ := appState.Store.ListBusinessCities(ctx)

		advisorNames := map[int64]string{}
		for i := range advisors {
			advisorNames[advisors[i].ID] = advisors[i].Name
		}

		zoneNames := map[int64]string{}
		for i := range zones {
			zoneNames[zones[i].ID] = zones[i].Name
		}

		businesses, err := appState.Store.ListBusinesses(ctx, tid, f)
		if err != nil {
			log.Error("b2b: negocios list", "error", err)
		}

		rows := make([]businessRow, 0, len(businesses))
		for i := range businesses {
			b := businesses[i]
			label, class := statusMeta(b.Status)
			row := businessRow{MapBusiness: b, StatusLabel: label, StatusClass: class}

			if b.AdvisorID != nil {
				row.AdvisorName = advisorNames[*b.AdvisorID]
			}

			if b.ZoneID != nil {
				row.ZoneName = zoneNames[*b.ZoneID]
			}

			rows = append(rows, row)
		}

		data := map[string]any{
			"Rows":     rows,
			"Count":    len(rows),
			"Advisors": advisors,
			"Cities":   cities,
			"QVal":     f.Search,
			"CityVal":  f.City,
			"StatVal":  f.Status,
			"AdvVal":   q.Get("advisor"),
			"Success":  q.Get("success"),
			"Error":    q.Get("error"),
		}

		renderTemplate(appState, w, r, "negocios.html", data)
	}
}

// AsesoresPageHandler renders the advisors management page.
func AsesoresPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		tid, _ := effectiveTenant(appState, r)

		advisors, err := appState.Store.ListAdvisors(r.Context(), tid)
		if err != nil {
			log.Error("b2b: asesores list", "error", err)
		}

		data := map[string]any{
			"Advisors": advisors,
			"Success":  r.URL.Query().Get("success"),
			"Error":    r.URL.Query().Get("error"),
		}

		renderTemplate(appState, w, r, "asesores.html", data)
	}
}

// zoneRow is a zone enriched with its advisor's name for the list view.
type zoneRow struct {
	Zone

	AdvisorName string
}

// ZonasPageHandler renders the zones management page.
func ZonasPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		ctx := r.Context()
		tid, _ := effectiveTenant(appState, r)

		advisors, _ := appState.Store.ListAdvisors(ctx, tid)

		advisorNames := map[int64]string{}
		for i := range advisors {
			advisorNames[advisors[i].ID] = advisors[i].Name
		}

		zones, err := appState.Store.ListZones(ctx, tid)
		if err != nil {
			log.Error("b2b: zonas list", "error", err)
		}

		rows := make([]zoneRow, 0, len(zones))
		for i := range zones {
			z := zones[i]
			row := zoneRow{Zone: z}

			if z.AdvisorID != nil {
				row.AdvisorName = advisorNames[*z.AdvisorID]
			}

			rows = append(rows, row)
		}

		data := map[string]any{
			"Rows":     rows,
			"Advisors": advisors,
			"Success":  r.URL.Query().Get("success"),
			"Error":    r.URL.Query().Get("error"),
		}

		renderTemplate(appState, w, r, "zonas.html", data)
	}
}
