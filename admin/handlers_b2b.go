package admin

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/gosom/google-maps-scraper/log"
	"github.com/gosom/google-maps-scraper/rqueue"
)

// B2BPageHandler renders the B2B map dashboard shell (advisors, zones, filters).
// The businesses themselves are loaded asynchronously via B2BBusinessesHandler.
func B2BPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		ctx := r.Context()

		advisors, err := appState.Store.ListAdvisors(ctx)
		if err != nil {
			log.Error("b2b: list advisors", "error", err)
		}

		zones, err := appState.Store.ListZones(ctx)
		if err != nil {
			log.Error("b2b: list zones", "error", err)
		}

		cities, err := appState.Store.ListBusinessCities(ctx)
		if err != nil {
			log.Error("b2b: list cities", "error", err)
		}

		summary, err := appState.Store.B2BSummary(ctx)
		if err != nil {
			log.Error("b2b: summary", "error", err)
			summary = &B2BSummary{}
		}

		data := map[string]any{
			"Advisors": advisors,
			"Zones":    zones,
			"Cities":   cities,
			"Summary":  summary,
			// Passed into a <script> context; html/template JSON-encodes these
			// safely for the map colouring logic.
			"AdvisorsData": advisors,
			"ZonesData":    zones,
			"Success":      r.URL.Query().Get("success"),
			"Error":        r.URL.Query().Get("error"),
		}

		renderTemplate(appState, w, r, "b2b.html", data)
	}
}

// B2BBusinessesHandler returns scraped businesses (with CRM overlay) as JSON,
// filtered by the query parameters city / status / advisor.
func B2BBusinessesHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		f := BusinessFilter{
			City:   r.URL.Query().Get("city"),
			Status: r.URL.Query().Get("status"),
			Search: strings.TrimSpace(r.URL.Query().Get("q")),
		}

		if a := strings.TrimSpace(r.URL.Query().Get("advisor")); a != "" {
			if id, err := strconv.ParseInt(a, 10, 64); err == nil {
				f.AdvisorID = &id
			}
		}

		businesses, err := appState.Store.ListBusinesses(r.Context(), f)
		if err != nil {
			log.Error("b2b: list businesses", "error", err)
			http.Error(w, "failed to load businesses", http.StatusInternalServerError)

			return
		}

		if businesses == nil {
			businesses = []MapBusiness{}
		}

		writeJSON(w, http.StatusOK, businesses)
	}
}

// B2BSummaryHandler returns the aggregate CRM counters as JSON so the KPI tiles
// can update live without a full page reload.
func B2BSummaryHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		sum, err := appState.Store.B2BSummary(r.Context())
		if err != nil {
			log.Error("b2b: summary json", "error", err)
			http.Error(w, "failed to load summary", http.StatusInternalServerError)

			return
		}

		writeJSON(w, http.StatusOK, map[string]int{
			"total":       sum.Total,
			"clients":     sum.Clients,
			"prospects":   sum.Prospects,
			"in_progress": sum.InProgress,
			"advisors":    sum.Advisors,
			"zones":       sum.Zones,
		})
	}
}

// b2bJobView is the compact job status returned to the dashboard for live
// progress of scrape searches.
type b2bJobView struct {
	JobID       string `json:"job_id"`
	Keyword     string `json:"keyword"`
	Status      string `json:"status"`
	ResultCount int    `json:"result_count"`
	Error       string `json:"error"`
}

// B2BJobsHandler returns the most recent scrape jobs as JSON so the dashboard
// can show live progress of running searches.
func B2BJobsHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		out := []b2bJobView{}

		if appState.RQueueClient != nil {
			result, err := appState.RQueueClient.ListJobs(r.Context(), "", 8, "")
			if err != nil {
				log.Error("b2b: list jobs", "error", err)
				http.Error(w, "failed to list jobs", http.StatusInternalServerError)

				return
			}

			for i := range result.Jobs {
				j := &result.Jobs[i]
				out = append(out, b2bJobView{
					JobID:       j.JobID,
					Keyword:     j.Keyword,
					Status:      j.Status,
					ResultCount: j.ResultCount,
					Error:       j.Error,
				})
			}
		}

		writeJSON(w, http.StatusOK, out)
	}
}

// b2bStatusRequest is the JSON payload for updating a business CRM overlay.
type b2bStatusRequest struct {
	Key       string `json:"key"`
	Status    string `json:"status"`
	AdvisorID *int64 `json:"advisor_id"`
	ZoneID    *int64 `json:"zone_id"`
	Notes     string `json:"notes"`
	Title     string `json:"title"`
}

// B2BSetStatusHandler upserts the CRM overlay for a single business.
func B2BSetStatusHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var req b2bStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}

		req.Key = strings.TrimSpace(req.Key)
		if req.Key == "" {
			http.Error(w, "missing business key", http.StatusBadRequest)
			return
		}

		if req.Status == "" {
			req.Status = StatusProspect
		}

		if !ValidBusinessStatus(req.Status) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}

		if err := appState.Store.SetBusinessCRM(
			r.Context(), req.Key, req.Status, req.AdvisorID, req.ZoneID, req.Notes, req.Title,
		); err != nil {
			log.Error("b2b: set business crm", "error", err, "key", req.Key)
			http.Error(w, "failed to save", http.StatusInternalServerError)

			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": req.Status})
	}
}

// B2BSearchHandler enqueues a Google Maps scrape job from the dashboard, so a
// non-technical user can launch a search ("restaurantes en Usaquén, Bogotá")
// with one click instead of calling the REST API. A running worker is required
// to actually process the queued job.
func B2BSearchHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		if appState.RQueueClient == nil {
			http.Redirect(w, r, "/admin/b2b?error=La+cola+de+trabajos+no+esta+disponible", http.StatusSeeOther)
			return
		}

		what := strings.TrimSpace(r.FormValue("what"))
		where := strings.TrimSpace(r.FormValue("where"))

		if what == "" {
			http.Redirect(w, r, "/admin/b2b?error=Escribe+que+buscar+(ej.+restaurantes)", http.StatusSeeOther)
			return
		}

		keyword := what
		if where != "" {
			keyword = what + " en " + where
		}

		maxDepth := 10
		if d, err := strconv.Atoi(strings.TrimSpace(r.FormValue("max_depth"))); err == nil && d > 0 {
			maxDepth = d
		}

		jobID, err := appState.RQueueClient.InsertJob(r.Context(), rqueue.ScrapeJobArgs{
			Keyword:  keyword,
			Lang:     "es",
			MaxDepth: maxDepth,
		})
		if err != nil {
			log.Error("b2b: enqueue search", "error", err, "keyword", keyword)
			http.Redirect(w, r, "/admin/b2b?error=No+se+pudo+encolar+la+busqueda", http.StatusSeeOther)

			return
		}

		msg := url.QueryEscape("Búsqueda encolada: \"" + keyword + "\" (job " + jobID + "). Necesitas un worker activo para procesarla; los negocios aparecerán en el mapa al terminar.")
		http.Redirect(w, r, "/admin/b2b?success="+msg, http.StatusSeeOther)
	}
}

// CreateAdvisorHandler handles the advisor creation form.
func CreateAdvisorHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			b2bRedirectBack(w, r, "/admin/b2b", "error", "El+nombre+del+asesor+es+obligatorio")
			return
		}

		_, err := appState.Store.CreateAdvisor(
			r.Context(), name,
			strings.TrimSpace(r.FormValue("email")),
			strings.TrimSpace(r.FormValue("phone")),
			strings.TrimSpace(r.FormValue("city")),
		)
		if err != nil {
			log.Error("b2b: create advisor", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b", "error", "No+se+pudo+crear+el+asesor")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b", "success", "Asesor+creado")
	}
}

// DeleteAdvisorHandler removes an advisor.
func DeleteAdvisorHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b", "error", "ID+invalido")
			return
		}

		if err := appState.Store.DeleteAdvisor(r.Context(), id); err != nil {
			log.Error("b2b: delete advisor", "error", err, "id", id)
			b2bRedirectBack(w, r, "/admin/b2b", "error", "No+se+pudo+eliminar+el+asesor")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b", "success", "Asesor+eliminado")
	}
}

// CreateZoneHandler handles the zone creation form.
func CreateZoneHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		city := strings.TrimSpace(r.FormValue("city"))

		if name == "" || city == "" {
			b2bRedirectBack(w, r, "/admin/b2b", "error", "Nombre+y+ciudad+de+la+zona+son+obligatorios")
			return
		}

		var advisorID *int64
		if a := strings.TrimSpace(r.FormValue("advisor_id")); a != "" {
			if id, err := strconv.ParseInt(a, 10, 64); err == nil {
				advisorID = &id
			}
		}

		if _, err := appState.Store.CreateZone(r.Context(), name, city, advisorID, strings.TrimSpace(r.FormValue("color"))); err != nil {
			log.Error("b2b: create zone", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b", "error", "No+se+pudo+crear+la+zona")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b", "success", "Zona+creada")
	}
}

// DeleteZoneHandler removes a zone.
func DeleteZoneHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b", "error", "ID+invalido")
			return
		}

		if err := appState.Store.DeleteZone(r.Context(), id); err != nil {
			log.Error("b2b: delete zone", "error", err, "id", id)
			b2bRedirectBack(w, r, "/admin/b2b", "error", "No+se+pudo+eliminar+la+zona")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b", "success", "Zona+eliminada")
	}
}

// writeJSON is a small helper to emit a JSON response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
