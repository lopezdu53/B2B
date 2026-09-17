package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"

	"github.com/gosom/google-maps-scraper/log"
)

const zoneImportMaxBytes = 8 << 20

var zoneCSVHeader = []string{
	"Key", "Negocio", "Categoría (Maps)", "Especialidad", "Dirección", "Ciudad",
	"Teléfono", "Web", "Maps", "Email", "Calificación", "N.º reseñas",
	"Estado", "Asesor", "Zona", "Categoría", "Notas",
}

// zoneShareToken mints an unguessable public link of the form
// "z.<tenantID>.<zoneID>.<hmac>" so a visitor can see only that zone.
func zoneShareToken(tenantID, zoneID int64, secret []byte) string {
	payload := strconv.FormatInt(tenantID, 10) + "." + strconv.FormatInt(zoneID, 10)
	h := hmac.New(sha256.New, secret)
	h.Write([]byte("zone-embed:" + payload))

	return "z." + payload + "." + hex.EncodeToString(h.Sum(nil))
}

// parseZoneShareToken validates a public zone token and returns tenant + zone ids.
func parseZoneShareToken(token string, secret []byte) (tenantID, zoneID int64, ok bool) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "z.") {
		return 0, 0, false
	}

	rest := token[2:]
	dot := strings.LastIndexByte(rest, '.')
	if dot <= 0 || dot == len(rest)-1 {
		return 0, 0, false
	}

	payload := rest[:dot]
	parts := strings.Split(payload, ".")
	if len(parts) != 2 {
		return 0, 0, false
	}

	tid, err1 := strconv.ParseInt(parts[0], 10, 64)
	zid, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil || tid <= 0 || zid <= 0 {
		return 0, 0, false
	}

	expected := zoneShareToken(tid, zid, secret)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return 0, 0, false
	}

	return tid, zid, true
}

func requestOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}

	proto := "http"
	if r.TLS != nil {
		proto = "https"
	}

	if p := firstHeaderValue(r.Header.Get("X-Forwarded-Proto")); p != "" {
		proto = p
	}

	host := firstHeaderValue(r.Header.Get("X-Forwarded-Host"))
	if host == "" {
		host = r.Host
	}

	if host == "" {
		return ""
	}

	return proto + "://" + host
}

func firstHeaderValue(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	if i := strings.IndexByte(raw, ','); i >= 0 {
		raw = strings.TrimSpace(raw[:i])
	}

	return raw
}

func zonePublicPath(tenantID, zoneID int64, secret []byte) string {
	return "/embed/zona?t=" + zoneShareToken(tenantID, zoneID, secret)
}

func zonePublicURL(r *http.Request, tenantID, zoneID int64, secret []byte) string {
	path := zonePublicPath(tenantID, zoneID, secret)
	if origin := requestOrigin(r); origin != "" {
		return origin + path
	}

	return path
}

// groupShareToken mints a public link of the form "g.<tenantID>.<groupID>.<hmac>".
func groupShareToken(tenantID, groupID int64, secret []byte) string {
	payload := strconv.FormatInt(tenantID, 10) + "." + strconv.FormatInt(groupID, 10)
	h := hmac.New(sha256.New, secret)
	h.Write([]byte("group-embed:" + payload))

	return "g." + payload + "." + hex.EncodeToString(h.Sum(nil))
}

// parseGroupShareToken validates a public group token and returns tenant + group ids.
func parseGroupShareToken(token string, secret []byte) (tenantID, groupID int64, ok bool) {
	token = strings.TrimSpace(token)
	if !strings.HasPrefix(token, "g.") {
		return 0, 0, false
	}

	rest := token[2:]
	dot := strings.LastIndexByte(rest, '.')
	if dot <= 0 || dot == len(rest)-1 {
		return 0, 0, false
	}

	payload := rest[:dot]
	parts := strings.Split(payload, ".")
	if len(parts) != 2 {
		return 0, 0, false
	}

	tid, err1 := strconv.ParseInt(parts[0], 10, 64)
	gid, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil || tid <= 0 || gid <= 0 {
		return 0, 0, false
	}

	expected := groupShareToken(tid, gid, secret)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return 0, 0, false
	}

	return tid, gid, true
}

func groupPublicPath(tenantID, groupID int64, secret []byte) string {
	return "/embed/grupo?t=" + groupShareToken(tenantID, groupID, secret)
}

func groupPublicURL(r *http.Request, tenantID, groupID int64, secret []byte) string {
	path := groupPublicPath(tenantID, groupID, secret)
	if origin := requestOrigin(r); origin != "" {
		return origin + path
	}

	return path
}

func listGroupBusinesses(appState *AppState, r *http.Request, tid int64, zones []Zone) ([]MapBusiness, error) {
	list, err := appState.Store.ListBusinesses(r.Context(), tid, BusinessFilter{})
	if err != nil {
		return nil, err
	}

	return FilterBusinessesInZones(list, zones), nil
}

func loadZoneForTenant(appState *AppState, r *http.Request, id int64) (*Zone, int64, error) {
	tid, _ := effectiveTenant(appState, r)

	z, err := appState.Store.GetZone(r.Context(), tid, id)
	if err != nil {
		return nil, tid, err
	}

	return z, tid, nil
}

func listZoneBusinesses(appState *AppState, r *http.Request, tid int64, z *Zone) ([]MapBusiness, error) {
	list, err := appState.Store.ListBusinesses(r.Context(), tid, BusinessFilter{})
	if err != nil {
		return nil, err
	}

	return FilterBusinessesInZone(list, z), nil
}

func zoneLookupNames(appState *AppState, r *http.Request, tid int64) (advisors, zones, categories map[int64]string) {
	advisors = map[int64]string{}
	zones = map[int64]string{}
	categories = map[int64]string{}

	if list, err := appState.Store.ListAdvisors(r.Context(), tid); err == nil {
		for i := range list {
			advisors[list[i].ID] = list[i].Name
		}
	}

	if list, err := appState.Store.ListZones(r.Context(), tid); err == nil {
		for i := range list {
			zones[list[i].ID] = list[i].Name
		}
	}

	if list, err := EnsureFixedCategories(r.Context(), appState.Store, tid); err == nil {
		for i := range list {
			categories[list[i].ID] = list[i].Name
		}
	}

	return advisors, zones, categories
}

func zoneFileSlug(name string) string {
	var b strings.Builder
	prevDash := true

	for _, r := range strings.ToLower(name) {
		switch r {
		case 'á', 'à':
			r = 'a'
		case 'é', 'è':
			r = 'e'
		case 'í', 'ì':
			r = 'i'
		case 'ó', 'ò':
			r = 'o'
		case 'ú', 'ù', 'ü':
			r = 'u'
		case 'ñ':
			r = 'n'
		}

		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false

			continue
		}

		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}

	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "zona"
	}

	return s
}

func writeZoneBusinessesCSV(w http.ResponseWriter, filename string, businesses []MapBusiness, advisorNames, zoneNames, categoryNames map[int64]string) {
	if filename == "" {
		filename = "zona.csv"
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write(zoneCSVHeader)

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
			b.Key, b.Title, b.Category, b.Specialty, b.Address, b.City, b.Phone, b.Website,
			b.MapsURL, b.Email, rating, reviews, label, advisorName, zoneName, categoryName, b.Notes,
		})
	}

	cw.Flush()
}

type zoneImportRow struct {
	Key   string
	Title string
}

func parseBusinessImportCSV(r io.Reader) ([]zoneImportRow, error) {
	raw, err := io.ReadAll(io.LimitReader(r, zoneImportMaxBytes+1))
	if err != nil {
		return nil, err
	}

	if len(raw) > zoneImportMaxBytes {
		return nil, fmt.Errorf("el archivo supera 8 MB")
	}

	text := strings.TrimPrefix(string(raw), "\uFEFF")
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("el archivo está vacío")
	}

	firstLine, _, _ := strings.Cut(text, "\n")
	comma := strings.Count(firstLine, ",")
	semi := strings.Count(firstLine, ";")
	delim := ','
	if semi > comma {
		delim = ';'
	}

	cr := csv.NewReader(strings.NewReader(text))
	cr.Comma = delim
	cr.LazyQuotes = true
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("no se pudo leer el CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("el Excel no tiene filas de negocios")
	}

	keyIdx, titleIdx := -1, -1
	for i, h := range records[0] {
		switch foldCSVHeader(h) {
		case "key", "place_id", "placeid", "cid", "clave":
			if keyIdx < 0 {
				keyIdx = i
			}
		case "negocio", "title", "nombre", "name":
			if titleIdx < 0 {
				titleIdx = i
			}
		}
	}

	if keyIdx < 0 && titleIdx < 0 {
		return nil, fmt.Errorf("el Excel necesita una columna Key o Negocio")
	}

	out := make([]zoneImportRow, 0, len(records)-1)
	for _, rec := range records[1:] {
		row := zoneImportRow{}
		if keyIdx >= 0 && keyIdx < len(rec) {
			row.Key = strings.TrimSpace(rec[keyIdx])
		}

		if titleIdx >= 0 && titleIdx < len(rec) {
			row.Title = strings.TrimSpace(rec[titleIdx])
		}

		if row.Key == "" && row.Title == "" {
			continue
		}

		out = append(out, row)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no hay filas con Key o Negocio")
	}

	return out, nil
}

func foldCSVHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ".", "")

	return s
}

func matchImportedBusinessKeys(rows []zoneImportRow, businesses []MapBusiness) (keys []string, skipped int) {
	byKey := make(map[string]string, len(businesses))
	byTitle := make(map[string]string, len(businesses))

	for i := range businesses {
		k := strings.TrimSpace(businesses[i].Key)
		if k == "" {
			continue
		}

		byKey[k] = k
		if t := cityKey(businesses[i].Title); t != "" {
			if _, exists := byTitle[t]; !exists {
				byTitle[t] = k
			}
		}
	}

	seen := map[string]struct{}{}
	for _, row := range rows {
		key := strings.TrimSpace(row.Key)
		if key != "" {
			if matched, ok := byKey[key]; ok {
				if _, dup := seen[matched]; !dup {
					seen[matched] = struct{}{}

					keys = append(keys, matched)
				}

				continue
			}
		}

		if t := cityKey(row.Title); t != "" {
			if matched, ok := byTitle[t]; ok {
				if _, dup := seen[matched]; !dup {
					seen[matched] = struct{}{}

					keys = append(keys, matched)
				}

				continue
			}
		}

		skipped++
	}

	return keys, skipped
}

// ZoneExportHandler streams the zone's businesses as an Excel-friendly CSV.
func ZoneExportHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			http.Error(w, "zona inválida", http.StatusBadRequest)
			return
		}

		z, tid, err := loadZoneForTenant(appState, r, id)
		if err != nil {
			if errors.Is(err, ErrResourceNotFound) {
				http.NotFound(w, r)
				return
			}

			log.Error("b2b: zone export get", "error", err)
			http.Error(w, "no se pudo exportar", http.StatusInternalServerError)

			return
		}

		businesses, err := listZoneBusinesses(appState, r, tid, z)
		if err != nil {
			log.Error("b2b: zone export list", "error", err)
			http.Error(w, "no se pudo exportar", http.StatusInternalServerError)

			return
		}

		advisors, zones, categories := zoneLookupNames(appState, r, tid)
		writeZoneBusinessesCSV(w, "zona-"+zoneFileSlug(z.Name)+".csv", businesses, advisors, zones, categories)
	}
}

// ZoneImportHandler assigns businesses from an uploaded CSV to the zone.
func ZoneImportHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Zona+invalida")
			return
		}

		if err := r.ParseMultipartForm(zoneImportMaxBytes); err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+leer+el+archivo")
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Selecciona+un+Excel+o+CSV")
			return
		}
		defer file.Close()

		rows, err := parseBusinessImportCSV(file)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", url.QueryEscape(err.Error()))
			return
		}

		z, tid, err := loadZoneForTenant(appState, r, id)
		if err != nil {
			if errors.Is(err, ErrResourceNotFound) {
				b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Zona+no+encontrada")
				return
			}

			log.Error("b2b: zone import get", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+importar")

			return
		}

		all, err := appState.Store.ListBusinesses(r.Context(), tid, BusinessFilter{})
		if err != nil {
			log.Error("b2b: zone import list", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+importar")

			return
		}

		keys, skipped := matchImportedBusinessKeys(rows, all)
		if len(keys) == 0 {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Ningun+negocio+coincidio+con+los+leads")
			return
		}

		if err := appState.Store.BulkSetZone(r.Context(), tid, keys, z.ID); err != nil {
			log.Error("b2b: zone import assign", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+asignar+la+zona")

			return
		}

		msg := strconv.Itoa(len(keys)) + "+negocios+asignados+a+" + url.QueryEscape(z.Name)
		if skipped > 0 {
			msg += "+(" + strconv.Itoa(skipped) + "+filas+sin+coincidencia)"
		}

		b2bRedirectBack(w, r, "/admin/b2b/zonas", "success", msg)
	}
}

func resolveZoneShare(appState *AppState, r *http.Request) (*Zone, int64, bool) {
	tid, zid, ok := parseZoneShareToken(r.URL.Query().Get("t"), appState.EncryptionKey)
	if !ok {
		return nil, 0, false
	}

	z, err := appState.Store.GetZone(r.Context(), tid, zid)
	if err != nil {
		return nil, 0, false
	}

	return z, tid, true
}

// EmbedZoneHandler renders the public, read-only page for a single zone.
func EmbedZoneHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		z, _, ok := resolveZoneShare(appState, r)
		if !ok {
			http.Error(w, "Enlace de zona inválido o vencido.", http.StatusForbidden)
			return
		}

		data := map[string]any{
			"Token":            r.URL.Query().Get("t"),
			"Zone":             z,
			"ZoneData":         asJSON([]Zone{*z}),
			"GoogleMapsAPIKey": resolveGoogleMapsAPIKey(appState, r),
			"AssetVersion":     assetVersion,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := appState.Templates.ExecuteTemplate(w, "embed_zona.html", data); err != nil {
			log.Error("embed: render zona", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// EmbedZoneBusinessesHandler returns JSON of the businesses that belong to the
// shared zone (assigned or inside the polygon).
func EmbedZoneBusinessesHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		z, tid, ok := resolveZoneShare(appState, r)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		businesses, err := listZoneBusinesses(appState, r, tid, z)
		if err != nil {
			log.Error("embed: zone businesses", "error", err)
			http.Error(w, "failed to load businesses", http.StatusInternalServerError)

			return
		}

		if businesses == nil {
			businesses = []MapBusiness{}
		}

		writeJSON(w, http.StatusOK, businesses)
	}
}

func resolveGroupShare(appState *AppState, r *http.Request) (*ZoneGroup, int64, bool) {
	tid, gid, ok := parseGroupShareToken(r.URL.Query().Get("t"), appState.EncryptionKey)
	if !ok {
		return nil, 0, false
	}

	g, err := appState.Store.GetZoneGroup(r.Context(), tid, gid)
	if err != nil {
		return nil, 0, false
	}

	return g, tid, true
}

// EmbedGroupHandler renders the public, read-only page for a zone group.
func EmbedGroupHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g, _, ok := resolveGroupShare(appState, r)
		if !ok {
			http.Error(w, "Enlace de grupo inválido o vencido.", http.StatusForbidden)
			return
		}

		zones := g.Zones
		if zones == nil {
			zones = []Zone{}
		}

		data := map[string]any{
			"Token":            r.URL.Query().Get("t"),
			"Group":            g,
			"ZoneData":         asJSON(zones),
			"GoogleMapsAPIKey": resolveGoogleMapsAPIKey(appState, r),
			"AssetVersion":     assetVersion,
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := appState.Templates.ExecuteTemplate(w, "embed_grupo.html", data); err != nil {
			log.Error("embed: render grupo", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// EmbedGroupBusinessesHandler returns JSON of businesses that belong to any
// zone in the shared group (assigned or inside a polygon).
func EmbedGroupBusinessesHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		g, tid, ok := resolveGroupShare(appState, r)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		businesses, err := listGroupBusinesses(appState, r, tid, g.Zones)
		if err != nil {
			log.Error("embed: group businesses", "error", err)
			http.Error(w, "failed to load businesses", http.StatusInternalServerError)

			return
		}

		if businesses == nil {
			businesses = []MapBusiness{}
		}

		writeJSON(w, http.StatusOK, businesses)
	}
}
