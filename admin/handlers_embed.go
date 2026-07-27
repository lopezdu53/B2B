package admin

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"github.com/gosom/google-maps-scraper/log"
)

// Embeddable, read-only map for external presentations (e.g. PowerPoint's
// "Web Viewer" add-in). These routes are NOT behind session auth: access is
// gated by a per-tenant HMAC token that only the server can mint, and they
// expose read-only data (no CRM editing).

// embedToken mints an unguessable, per-tenant token of the form
// "<tenantID>.<hmac>". The HMAC is keyed by the server's encryption key, so a
// token cannot be forged without it.
func embedToken(tenantID int64, secret []byte) string {
	id := strconv.FormatInt(tenantID, 10)
	h := hmac.New(sha256.New, secret)
	h.Write([]byte("embed:" + id))

	return id + "." + hex.EncodeToString(h.Sum(nil))
}

// parseEmbedToken validates a token and returns its tenant id.
func parseEmbedToken(token string, secret []byte) (int64, bool) {
	dot := strings.LastIndexByte(token, '.')
	if dot <= 0 || dot == len(token)-1 {
		return 0, false
	}

	idPart, sig := token[:dot], token[dot+1:]

	id, err := strconv.ParseInt(idPart, 10, 64)
	if err != nil {
		return 0, false
	}

	expected := embedToken(id, secret)
	if !hmac.Equal([]byte(token), []byte(expected)) {
		return 0, false
	}

	_ = sig

	return id, true
}

// EmbedMapHandler renders the standalone, framable map page.
func EmbedMapHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid, ok := parseEmbedToken(r.URL.Query().Get("t"), appState.EncryptionKey)
		if !ok {
			http.Error(w, "Enlace de incrustación inválido o vencido.", http.StatusForbidden)
			return
		}

		ctx := r.Context()

		advisors, _ := appState.Store.ListAdvisors(ctx, tid)
		zones, _ := appState.Store.ListZones(ctx, tid)
		categories, _ := appState.Store.ListCategories(ctx, tid)
		cities, _ := appState.Store.ListBusinessCities(ctx)

		data := map[string]any{
			"Token":          r.URL.Query().Get("t"),
			"Advisors":       advisors,
			"Zones":          zones,
			"Categories":     categories,
			"Cities":         cities,
			"AdvisorsData":   advisors,
			"ZonesData":      zones,
			"CategoriesData": categories,
		}

		// Rendered directly (not via renderTemplate) so no session/navbar
		// context is required, and no frame-blocking headers are set.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if err := appState.Templates.ExecuteTemplate(w, "embed_map.html", data); err != nil {
			log.Error("embed: render map", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

// EmbedBusinessesHandler returns the tenant's businesses as JSON for the embed
// map, filtered by city / status / advisor.
func EmbedBusinessesHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tid, ok := parseEmbedToken(r.URL.Query().Get("t"), appState.EncryptionKey)
		if !ok {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

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

		if c := strings.TrimSpace(q.Get("category")); c != "" {
			if id, err := strconv.ParseInt(c, 10, 64); err == nil {
				f.CategoryID = &id
			}
		}

		businesses, err := appState.Store.ListBusinesses(r.Context(), tid, f)
		if err != nil {
			log.Error("embed: list businesses", "error", err)
			http.Error(w, "failed to load businesses", http.StatusInternalServerError)

			return
		}

		if businesses == nil {
			businesses = []MapBusiness{}
		}

		writeJSON(w, http.StatusOK, businesses)
	}
}
