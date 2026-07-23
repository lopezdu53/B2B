package admin

import (
	"encoding/csv"
	"net/http"
	"net/url"
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

// businessRow is a scraped business enriched with resolved advisor/zone/category
// names for the list view.
type businessRow struct {
	MapBusiness

	StatusLabel  string
	StatusClass  string
	AdvisorName  string
	ZoneName     string
	CategoryName string
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
			Sort:   q.Get("sort"),
		}

		if a := strings.TrimSpace(q.Get("advisor")); a != "" {
			if id, err := strconv.ParseInt(a, 10, 64); err == nil {
				f.AdvisorID = &id
			}
		}

		if c := strings.TrimSpace(q.Get("category")); c != "" {
			if id, err := strconv.ParseInt(c, 10, 64); err == nil {
				f.CategoryID = &id
			}
		}

		// Advisor users only ever see their own assigned businesses.
		if scope := advisorScope(r); scope != nil {
			f.AdvisorID = scope
		}

		// Pagination.
		const pageSize = 50

		page, _ := strconv.Atoi(q.Get("page"))
		if page < 1 {
			page = 1
		}

		f.Limit = pageSize
		f.Offset = (page - 1) * pageSize

		tid, _ := effectiveTenant(appState, r)

		total, _ := appState.Store.CountBusinesses(ctx, tid, f)

		totalPages := (total + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}

		// Preserve filters across page links.
		params := url.Values{}
		for _, kv := range []struct{ k, v string }{
			{"q", f.Search}, {"city", f.City}, {"status", f.Status},
			{"advisor", q.Get("advisor")}, {"category", q.Get("category")}, {"sort", f.Sort},
		} {
			if kv.v != "" {
				params.Set(kv.k, kv.v)
			}
		}

		exportURL := "/admin/b2b/negocios/export?" + params.Encode()

		pageURL := func(p int) string {
			pp := url.Values{}
			for k, v := range params {
				pp[k] = v
			}

			pp.Set("page", strconv.Itoa(p))

			return "/admin/b2b/negocios?" + pp.Encode()
		}

		advisors, _ := appState.Store.ListAdvisors(ctx, tid)
		zones, _ := appState.Store.ListZones(ctx, tid)
		categories, _ := appState.Store.ListCategories(ctx, tid)
		cities, _ := appState.Store.ListBusinessCities(ctx)

		advisorNames := map[int64]string{}
		for i := range advisors {
			advisorNames[advisors[i].ID] = advisors[i].Name
		}

		zoneNames := map[int64]string{}
		for i := range zones {
			zoneNames[zones[i].ID] = zones[i].Name
		}

		categoryNames := map[int64]string{}
		for i := range categories {
			categoryNames[categories[i].ID] = categories[i].Name
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

			if b.CategoryID != nil {
				row.CategoryName = categoryNames[*b.CategoryID]
			}

			rows = append(rows, row)
		}

		data := map[string]any{
			"Rows":       rows,
			"Count":      total,
			"Advisors":   advisors,
			"Zones":      zones,
			"Categories": categories,
			"Cities":     cities,
			"QVal":       f.Search,
			"CityVal":    f.City,
			"StatVal":    f.Status,
			"AdvVal":     q.Get("advisor"),
			"CatVal":     q.Get("category"),
			"SortVal":    f.Sort,
			"Page":       page,
			"TotalPages": totalPages,
			"HasPrev":    page > 1,
			"HasNext":    page < totalPages,
			"PrevURL":    pageURL(page - 1),
			"NextURL":    pageURL(page + 1),
			"ExportURL":  exportURL,
			"IsAdvisor":  advisorScope(r) != nil,
			"Success":    q.Get("success"),
			"Error":      q.Get("error"),
		}

		renderTemplate(appState, w, r, "negocios.html", data)
	}
}

// NegociosExportHandler streams the filtered businesses as a CSV download.
func NegociosExportHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()

		f := BusinessFilter{City: q.Get("city"), Status: q.Get("status"), Search: strings.TrimSpace(q.Get("q"))}

		if a := strings.TrimSpace(q.Get("advisor")); a != "" {
			if id, err := strconv.ParseInt(a, 10, 64); err == nil {
				f.AdvisorID = &id
			}
		}

		if c := strings.TrimSpace(q.Get("category")); c != "" {
			if id, err := strconv.ParseInt(c, 10, 64); err == nil {
				f.CategoryID = &id
			}
		}

		if scope := advisorScope(r); scope != nil {
			f.AdvisorID = scope
		}

		tid, _ := effectiveTenant(appState, r)

		advisors, _ := appState.Store.ListAdvisors(ctx, tid)
		zones, _ := appState.Store.ListZones(ctx, tid)
		categories, _ := appState.Store.ListCategories(ctx, tid)

		advisorNames := map[int64]string{}
		for i := range advisors {
			advisorNames[advisors[i].ID] = advisors[i].Name
		}

		zoneNames := map[int64]string{}
		for i := range zones {
			zoneNames[zones[i].ID] = zones[i].Name
		}

		categoryNames := map[int64]string{}
		for i := range categories {
			categoryNames[categories[i].ID] = categories[i].Name
		}

		businesses, err := appState.Store.ListBusinesses(ctx, tid, f)
		if err != nil {
			log.Error("b2b: export", "error", err)
			http.Error(w, "export failed", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="negocios.csv"`)
		_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM for Excel

		cw := csv.NewWriter(w)
		_ = cw.Write([]string{"Negocio", "Categoría (Maps)", "Dirección", "Ciudad", "Teléfono", "Web", "Calificación", "N.º reseñas", "Estado", "Asesor", "Zona", "Categoría", "Notas"})

		for i := range businesses {
			b := businesses[i]
			label, _ := statusMeta(b.Status)

			var advisorName, zoneName, categoryName string
			if b.AdvisorID != nil {
				advisorName = advisorNames[*b.AdvisorID]
			}

			if b.ZoneID != nil {
				zoneName = zoneNames[*b.ZoneID]
			}

			if b.CategoryID != nil {
				categoryName = categoryNames[*b.CategoryID]
			}

			rating := ""
			reviews := ""

			if b.ReviewCount > 0 {
				rating = strconv.FormatFloat(b.Rating, 'f', 1, 64)
				reviews = strconv.Itoa(b.ReviewCount)
			}

			_ = cw.Write([]string{
				b.Title, b.Category, b.Address, b.City, b.Phone, b.Website,
				rating, reviews, label, advisorName, zoneName, categoryName, b.Notes,
			})
		}

		cw.Flush()
	}
}

// PapeleraPageHandler renders the trash: businesses the tenant sent to trash
// (hidden). From here they can be restored back into the active list.
func PapeleraPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		ctx := r.Context()
		q := r.URL.Query()

		f := BusinessFilter{
			Search: strings.TrimSpace(q.Get("q")),
			Hidden: true,
		}

		// Advisor users only ever see their own assigned businesses.
		if scope := advisorScope(r); scope != nil {
			f.AdvisorID = scope
		}

		const pageSize = 50

		page, _ := strconv.Atoi(q.Get("page"))
		if page < 1 {
			page = 1
		}

		f.Limit = pageSize
		f.Offset = (page - 1) * pageSize

		tid, _ := effectiveTenant(appState, r)

		total, _ := appState.Store.CountBusinesses(ctx, tid, f)

		totalPages := (total + pageSize - 1) / pageSize
		if totalPages < 1 {
			totalPages = 1
		}

		params := url.Values{}
		if f.Search != "" {
			params.Set("q", f.Search)
		}

		pageURL := func(p int) string {
			pp := url.Values{}
			for k, v := range params {
				pp[k] = v
			}

			pp.Set("page", strconv.Itoa(p))

			return "/admin/b2b/papelera?" + pp.Encode()
		}

		advisors, _ := appState.Store.ListAdvisors(ctx, tid)
		categories, _ := appState.Store.ListCategories(ctx, tid)

		advisorNames := map[int64]string{}
		for i := range advisors {
			advisorNames[advisors[i].ID] = advisors[i].Name
		}

		categoryNames := map[int64]string{}
		for i := range categories {
			categoryNames[categories[i].ID] = categories[i].Name
		}

		businesses, err := appState.Store.ListBusinesses(ctx, tid, f)
		if err != nil {
			log.Error("b2b: papelera list", "error", err)
		}

		rows := make([]businessRow, 0, len(businesses))
		for i := range businesses {
			b := businesses[i]
			label, class := statusMeta(b.Status)
			row := businessRow{MapBusiness: b, StatusLabel: label, StatusClass: class}

			if b.AdvisorID != nil {
				row.AdvisorName = advisorNames[*b.AdvisorID]
			}

			if b.CategoryID != nil {
				row.CategoryName = categoryNames[*b.CategoryID]
			}

			rows = append(rows, row)
		}

		data := map[string]any{
			"Rows":       rows,
			"Count":      total,
			"QVal":       f.Search,
			"Page":       page,
			"TotalPages": totalPages,
			"HasPrev":    page > 1,
			"HasNext":    page < totalPages,
			"PrevURL":    pageURL(page - 1),
			"NextURL":    pageURL(page + 1),
			"IsAdvisor":  advisorScope(r) != nil,
			"Success":    q.Get("success"),
			"Error":      q.Get("error"),
		}

		renderTemplate(appState, w, r, "papelera.html", data)
	}
}

// PapeleraBulkHandler restores selected businesses from the trash.
func PapeleraBulkHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/papelera", "error", "Formulario+invalido")
			return
		}

		keys := r.PostForm["keys"]
		if len(keys) == 0 {
			b2bRedirectBack(w, r, "/admin/b2b/papelera", "error", "Selecciona+al+menos+un+negocio")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.BulkSetHidden(r.Context(), tid, keys, false); err != nil {
			log.Error("b2b: papelera restore", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/papelera", "error", "No+se+pudo+restaurar")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/papelera", "success", strconv.Itoa(len(keys))+"+negocios+restaurados")
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
