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
		tid, _ := effectiveTenant(appState, r)

		advisors, err := appState.Store.ListAdvisors(ctx, tid)
		if err != nil {
			log.Error("b2b: list advisors", "error", err)
		}

		zones, err := appState.Store.ListZones(ctx, tid)
		if err != nil {
			log.Error("b2b: list zones", "error", err)
		}

		categories, err := appState.Store.ListCategories(ctx, tid)
		if err != nil {
			log.Error("b2b: list categories", "error", err)
		}

		cities := CityList()
		bogotaLocs := BogotaUrbanLocalidades()
		bogotaIdx, _ := LoadBogotaIndex()

		summary, err := appState.Store.B2BSummary(ctx, tid, advisorScope(r))
		if err != nil {
			log.Error("b2b: summary", "error", err)
			summary = &B2BSummary{}
		}

		data := map[string]any{
			"Advisors":          advisors,
			"Zones":             zones,
			"Categories":        categories,
			"Cities":            cities,
			"CityVal":           DefaultCity,
			"BogotaLocalidades": bogotaLocs,
			"BogotaIndexData":   asJSON(bogotaIdx),
			"SearchRubros":      SearchRubros(),
			"SearchRubrosData":  asJSON(SearchRubros()),
			"Summary":           summary,
			// JSON for the map colouring logic (template.JS, not Go dump).
			"AdvisorsData":   asJSON(advisors),
			"ZonesData":      asJSON(zones),
			"CategoriesData": asJSON(categories),
			// Embed (PowerPoint / external presentations): a per-tenant token
			// that unlocks the read-only /embed/map page. Only offered on the
			// tenant-wide view (not to advisor-scoped users).
			"EmbedToken": embedToken(tid, appState.EncryptionKey),
			"CanEmbed":   advisorScope(r) == nil,
			"Success":    r.URL.Query().Get("success"),
			"Error":      r.URL.Query().Get("error"),
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

		city, _ := ResolveCityFilter(r.URL.Query().Get("city"))
		f := BusinessFilter{
			City:   city,
			Status: r.URL.Query().Get("status"),
			Search: strings.TrimSpace(r.URL.Query().Get("q")),
		}

		if a := strings.TrimSpace(r.URL.Query().Get("advisor")); a != "" {
			if id, err := strconv.ParseInt(a, 10, 64); err == nil {
				f.AdvisorID = &id
			}
		}

		if c := strings.TrimSpace(r.URL.Query().Get("category")); c != "" {
			if id, err := strconv.ParseInt(c, 10, 64); err == nil {
				f.CategoryID = &id
			}
		}

		// Advisor users only ever see their own assigned businesses.
		if scope := advisorScope(r); scope != nil {
			f.AdvisorID = scope
		}

		tid, _ := effectiveTenant(appState, r)

		businesses, err := appState.Store.ListBusinesses(r.Context(), tid, f)
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

		tid, _ := effectiveTenant(appState, r)

		sum, err := appState.Store.B2BSummary(r.Context(), tid, advisorScope(r))
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

		// Advisors must not see the shared scrape queue (other tenants' keywords).
		if advisorScope(r) != nil {
			writeJSON(w, http.StatusOK, out)
			return
		}

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

		tid, _ := effectiveTenant(appState, r)

		if scope := advisorScope(r); scope != nil {
			req.AdvisorID = scope

			if err := ensureAdvisorOwns(appState, r, tid, []string{req.Key}); err != nil {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
		}

		if err := appState.Store.SetBusinessCRM(
			r.Context(), tid, req.Key, req.Status, req.AdvisorID, req.ZoneID, req.Notes, req.Title,
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

		terms, rubroLabel, ok := ResolveSearchTerms(r.FormValue("category"), r.FormValue("specialty"))
		if !ok || len(terms) == 0 {
			if what := strings.TrimSpace(r.FormValue("what")); what != "" {
				terms = []string{what}
				rubroLabel = what
			}
		}

		if len(terms) == 0 {
			http.Redirect(w, r, "/admin/b2b?error=Elige+que+buscar+(Restaurantes,+SuperMercados+o+Hoteles)", http.StatusSeeOther)
			return
		}

		maxDepth := DefaultSearchDepth
		if d, err := strconv.Atoi(strings.TrimSpace(r.FormValue("max_depth"))); err == nil && d > 0 {
			maxDepth = d
		}

		if maxDepth > MaxSearchDepth {
			maxDepth = MaxSearchDepth
		}

		city := r.FormValue("city")
		localidad := r.FormValue("localidad")
		barrio := r.FormValue("barrio")
		where := r.FormValue("where")

		queued := make([]string, 0, len(terms))
		keywords := make([]string, 0, len(terms))

		for _, term := range terms {
			keyword := SearchKeyword(term, city, localidad, barrio, where)
			jobID, err := appState.RQueueClient.InsertJob(r.Context(), rqueue.ScrapeJobArgs{
				Keyword:  keyword,
				Lang:     "es",
				MaxDepth: maxDepth,
			})
			if err != nil {
				log.Error("b2b: enqueue search", "error", err, "keyword", keyword)
				if len(queued) == 0 {
					http.Redirect(w, r, "/admin/b2b?error=No+se+pudo+encolar+la+busqueda", http.StatusSeeOther)

					return
				}

				break
			}

			queued = append(queued, jobID)
			keywords = append(keywords, keyword)
		}

		whereLabel := BuildSearchWhere(city, localidad, barrio)
		var msg string

		switch len(queued) {
		case 1:
			msg = url.QueryEscape("Búsqueda encolada: \"" + keywords[0] + "\" (job " + queued[0] + "). Necesitas un worker activo para procesarla; los negocios aparecerán en el mapa al terminar.")
		default:
			msg = url.QueryEscape("Se encolaron " + strconv.Itoa(len(queued)) + " búsquedas de " + rubroLabel + " en " + whereLabel + ". Un worker activo las procesará; los negocios aparecerán en el mapa al terminar.")
		}

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

		tid, _ := effectiveTenant(appState, r)

		advisor, err := appState.Store.CreateAdvisor(
			r.Context(), tid, name,
			strings.TrimSpace(r.FormValue("email")),
			strings.TrimSpace(r.FormValue("phone")),
			strings.TrimSpace(r.FormValue("city")),
		)
		if err != nil {
			log.Error("b2b: create advisor", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b", "error", "No+se+pudo+crear+el+asesor")

			return
		}

		// Optionally give the advisor a login (role=advisor) scoped to the tenant.
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		if username != "" || password != "" {
			if username == "" || !passwordMeetsPolicy(password) {
				b2bRedirectBack(w, r, "/admin/b2b", "error", "Usuario+y+contraseña+(mín.+8+caracteres)+son+obligatorios")
				return
			}
		}

		if username != "" && password != "" {
			user, uerr := appState.Store.CreateTenantUser(r.Context(), username, password, RoleAdvisor, tid)
			if uerr != nil {
				b2bRedirectBack(w, r, "/admin/b2b", "error", "Asesor+creado,+pero+el+acceso+falló+(usuario+en+uso)")
				return
			}

			if lerr := appState.Store.LinkUserAdvisor(r.Context(), user.ID, advisor.ID); lerr != nil {
				log.Error("b2b: link advisor user", "error", lerr)
			}

			b2bRedirectBack(w, r, "/admin/b2b", "success", "Asesor+y+acceso+creados")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b", "success", "Asesor+creado")
	}
}

// UpdateAdvisorHandler edits an advisor.
func UpdateAdvisorHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/asesores", "error", "ID+invalido")
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			b2bRedirectBack(w, r, "/admin/b2b/asesores", "error", "El+nombre+es+obligatorio")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.UpdateAdvisor(r.Context(), tid, id, name,
			strings.TrimSpace(r.FormValue("email")),
			strings.TrimSpace(r.FormValue("phone")),
			strings.TrimSpace(r.FormValue("city")),
		); err != nil {
			log.Error("b2b: update advisor", "error", err, "id", id)
			b2bRedirectBack(w, r, "/admin/b2b/asesores", "error", "No+se+pudo+actualizar")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/asesores", "success", "Asesor+actualizado")
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

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.DeleteAdvisor(r.Context(), tid, id); err != nil {
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

		tid, _ := effectiveTenant(appState, r)

		if _, err := appState.Store.CreateZone(r.Context(), tid, name, city, advisorID, strings.TrimSpace(r.FormValue("color"))); err != nil {
			log.Error("b2b: create zone", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b", "error", "No+se+pudo+crear+la+zona")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b", "success", "Zona+creada")
	}
}

// UpdateZoneHandler edits a zone.
func UpdateZoneHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "ID+invalido")
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		city := strings.TrimSpace(r.FormValue("city"))

		if name == "" || city == "" {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Nombre+y+ciudad+son+obligatorios")
			return
		}

		var advisorID *int64
		if a := strings.TrimSpace(r.FormValue("advisor_id")); a != "" {
			if aid, perr := strconv.ParseInt(a, 10, 64); perr == nil {
				advisorID = &aid
			}
		}

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.UpdateZone(r.Context(), tid, id, name, city, advisorID, strings.TrimSpace(r.FormValue("color"))); err != nil {
			log.Error("b2b: update zone", "error", err, "id", id)
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+actualizar")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/zonas", "success", "Zona+actualizada")
	}
}

// UpdateBusinessFormHandler updates a business CRM overlay from a normal form
// (used by the Negocios list page's edit modal), then redirects back.
func UpdateBusinessFormHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		key := strings.TrimSpace(r.FormValue("key"))
		status := r.FormValue("status")

		if key == "" {
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Negocio+invalido")
			return
		}

		if status == "" {
			status = StatusProspect
		}

		if !ValidBusinessStatus(status) {
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Estado+invalido")
			return
		}

		var advisorID, zoneID *int64
		if a := strings.TrimSpace(r.FormValue("advisor_id")); a != "" {
			if v, perr := strconv.ParseInt(a, 10, 64); perr == nil {
				advisorID = &v
			}
		}

		if z := strings.TrimSpace(r.FormValue("zone_id")); z != "" {
			if v, perr := strconv.ParseInt(z, 10, 64); perr == nil {
				zoneID = &v
			}
		}

		tid, _ := effectiveTenant(appState, r)

		if scope := advisorScope(r); scope != nil {
			advisorID = scope

			if err := ensureAdvisorOwns(appState, r, tid, []string{key}); err != nil {
				b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "No+autorizado")
				return
			}
		}

		if err := appState.Store.SetBusinessCRM(r.Context(), tid, key, status, advisorID, zoneID,
			strings.TrimSpace(r.FormValue("notes")), strings.TrimSpace(r.FormValue("title"))); err != nil {
			log.Error("b2b: update business form", "error", err, "key", key)
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "No+se+pudo+guardar")

			return
		}

		var categoryID *int64
		if c := strings.TrimSpace(r.FormValue("category_id")); c != "" {
			if v, perr := strconv.ParseInt(c, 10, 64); perr == nil {
				categoryID = &v
			}
		}

		if err := appState.Store.SetBusinessCategory(r.Context(), tid, key, categoryID); err != nil {
			log.Error("b2b: set business category", "error", err, "key", key)
		}

		b2bRedirectBack(w, r, "/admin/b2b/negocios", "success", "Negocio+actualizado")
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

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.DeleteZone(r.Context(), tid, id); err != nil {
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

// uniqueKeys returns trimmed, non-empty keys with duplicates removed.
func uniqueKeys(keys []string) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))

	for _, k := range keys {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}

		if _, ok := seen[k]; ok {
			continue
		}

		seen[k] = struct{}{}
		out = append(out, k)
	}

	return out
}

// ensureAdvisorOwns rejects the write when an advisor tries to mutate a
// business that is not assigned to them.
func ensureAdvisorOwns(appState *AppState, r *http.Request, tenantID int64, keys []string) error {
	scope := advisorScope(r)
	if scope == nil {
		return nil
	}

	keys = uniqueKeys(keys)
	if len(keys) == 0 {
		return errForbidden
	}

	owned, err := appState.Store.OwnedBusinessKeys(r.Context(), tenantID, *scope, keys)
	if err != nil {
		return err
	}

	if len(owned) != len(keys) {
		return errForbidden
	}

	return nil
}
