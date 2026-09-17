package admin

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/gosom/google-maps-scraper/log"
)

func parseZoneIDForm(r *http.Request) []int64 {
	if r == nil {
		return nil
	}

	vals := r.Form["zone_id"]
	if len(vals) == 0 {
		vals = r.PostForm["zone_id"]
	}

	seen := map[int64]struct{}{}
	out := make([]int64, 0, len(vals))

	for _, raw := range vals {
		id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil || id <= 0 {
			continue
		}

		if _, dup := seen[id]; dup {
			continue
		}

		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

func zoneGroupFormError(err error) string {
	if err == nil {
		return ""
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "nombre del grupo"):
		return "El+nombre+del+grupo+es+obligatorio"
	case strings.Contains(msg, "elige al menos"):
		return "Elige+al+menos+una+zona"
	default:
		return ""
	}
}

// CreateZoneGroupHandler creates a named group of zones.
func CreateZoneGroupHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		zoneIDs := parseZoneIDForm(r)

		if name == "" {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "El+nombre+del+grupo+es+obligatorio")
			return
		}

		if len(zoneIDs) == 0 {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Elige+al+menos+una+zona")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		if _, err := appState.Store.CreateZoneGroup(r.Context(), tid, name, zoneIDs); err != nil {
			if userErr := zoneGroupFormError(err); userErr != "" {
				b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", userErr)
				return
			}

			log.Error("b2b: create zone group", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+crear+el+grupo")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/zonas", "success", "Grupo+de+zonas+creado")
	}
}

// UpdateZoneGroupHandler edits a group's name and members.
func UpdateZoneGroupHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Grupo+invalido")
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		zoneIDs := parseZoneIDForm(r)

		if name == "" {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "El+nombre+del+grupo+es+obligatorio")
			return
		}

		if len(zoneIDs) == 0 {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Elige+al+menos+una+zona")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.UpdateZoneGroup(r.Context(), tid, id, name, zoneIDs); err != nil {
			if errors.Is(err, ErrResourceNotFound) {
				b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Grupo+no+encontrado")
				return
			}

			log.Error("b2b: update zone group", "error", err)
			if userErr := zoneGroupFormError(err); userErr != "" {
				b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", userErr)
				return
			}

			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+actualizar+el+grupo")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/zonas", "success", "Grupo+actualizado")
	}
}

// DeleteZoneGroupHandler removes a zone group.
func DeleteZoneGroupHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Grupo+invalido")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		if err := appState.Store.DeleteZoneGroup(r.Context(), tid, id); err != nil {
			if errors.Is(err, ErrResourceNotFound) {
				b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "Grupo+no+encontrado")
				return
			}

			log.Error("b2b: delete zone group", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/zonas", "error", "No+se+pudo+eliminar+el+grupo")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/zonas", "success", "Grupo+eliminado")
	}
}
