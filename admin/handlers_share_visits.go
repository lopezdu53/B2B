package admin

import (
	"context"
	"net/http"
	"time"

	"github.com/gosom/google-maps-scraper/log"
)

// VisitasPageHandler lists visitors of public zone and zone-group map links.
func VisitasPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		tid, _ := effectiveTenant(appState, r)

		visits, err := appState.Store.ListShareVisits(r.Context(), tid, 200)
		if err != nil {
			log.Error("b2b: visitas list", "error", err)
		}

		data := map[string]any{
			"Visits": visits,
		}

		renderTemplate(appState, w, r, "visitas.html", data)
	}
}

func recordShareVisit(appState *AppState, r *http.Request, tenantID int64, kind string, targetID int64, targetName string) ShareVisit {
	ip := clientIPFromRequest(r)
	ua := ""
	ctx := context.Background()
	hint := ""
	if r != nil {
		ua = r.UserAgent()
		ctx = r.Context()
		hint = r.Header.Get("CF-IPCountry")
	}
	place := LookupIPPlace(ctx, ip, hint)

	visit := newShareVisit(kind, targetID, targetName, ip, ua, place, time.Now())
	if appState == nil || appState.Store == nil || tenantID <= 0 {
		return visit
	}

	if err := appState.Store.CreateShareVisit(ctx, tenantID, visit); err != nil {
		log.Error("b2b: record share visit", "error", err)
	}

	return visit
}
