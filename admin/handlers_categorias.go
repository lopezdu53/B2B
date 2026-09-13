package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/gosom/google-maps-scraper/log"
)

// CategoriasPageHandler renders the categories management page.
func CategoriasPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		tid, _ := effectiveTenant(appState, r)

		cats, err := EnsureFixedCategories(r.Context(), appState.Store, tid)
		if err != nil {
			log.Error("b2b: categorias list", "error", err)
		}

		data := map[string]any{
			"Rows":    cats,
			"Success": r.URL.Query().Get("success"),
			"Error":   r.URL.Query().Get("error"),
		}

		renderTemplate(appState, w, r, "categorias.html", data)
	}
}

// CreateCategoryHandler rejects new categories: only the three locked ones exist.
func CreateCategoryHandler(_ *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "Las+categorías+son+fijas+(Restaurantes,+SúperMercados+y+Hoteles)")
	}
}

// UpdateCategoryHandler edits a category's name, color and icon.
func UpdateCategoryHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
		if err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "ID+invalido")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		cats, _ := appState.Store.ListCategories(r.Context(), tid)
		var current *Category
		for i := range cats {
			if cats[i].ID == id {
				current = &cats[i]
				break
			}
		}

		if current == nil || CanonicalFixedCategory(current.Name) == "" {
			b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "Solo+se+pueden+editar+las+3+categorías+principales")
			return
		}

		color := strings.TrimSpace(r.FormValue("color"))
		icon := strings.TrimSpace(r.FormValue("icon"))

		if err := appState.Store.UpdateCategory(r.Context(), tid, id, current.Name, color, icon); err != nil {
			log.Error("b2b: update category", "error", err, "id", id)
			b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "No+se+pudo+guardar")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/categorias", "success", "Categoría+actualizada")
	}
}

// DeleteCategoryHandler rejects deletes: the three principal categories are locked.
func DeleteCategoryHandler(_ *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "Las+categorías+principales+no+se+pueden+eliminar")
	}
}

// BulkBusinessHandler applies a CRM action to many selected businesses at once.
func BulkBusinessHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		if err := r.ParseForm(); err != nil {
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Formulario+invalido")
			return
		}

		keys := r.PostForm["keys"]
		action := r.FormValue("action")
		value := strings.TrimSpace(r.FormValue("value"))

		if len(keys) == 0 {
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Selecciona+al+menos+un+negocio")
			return
		}

		keys = uniqueKeys(keys)
		if len(keys) == 0 {
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Selecciona+al+menos+un+negocio")
			return
		}

		tid, _ := effectiveTenant(appState, r)
		ctx := r.Context()

		if scope := advisorScope(r); scope != nil {
			switch action {
			case "status", "delete":
				if err := ensureAdvisorOwns(appState, r, tid, keys); err != nil {
					b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "No+autorizado")
					return
				}
			default:
				b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Los+asesores+solo+pueden+cambiar+estado+o+enviar+a+papelera")
				return
			}
		}

		var err error

		switch action {
		case "status":
			err = appState.Store.BulkSetStatus(ctx, tid, keys, value)
		case "advisor":
			id, perr := strconv.ParseInt(value, 10, 64)
			if perr != nil {
				b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Elige+un+asesor")
				return
			}

			err = appState.Store.BulkSetAdvisor(ctx, tid, keys, id)
		case "zone":
			id, perr := strconv.ParseInt(value, 10, 64)
			if perr != nil {
				b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Elige+una+zona")
				return
			}

			err = appState.Store.BulkSetZone(ctx, tid, keys, id)
		case "category":
			id, perr := strconv.ParseInt(value, 10, 64)
			if perr != nil {
				b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Elige+una+categoría")
				return
			}

			err = appState.Store.BulkSetCategory(ctx, tid, keys, id)
		case "delete":
			err = appState.Store.BulkSetHidden(ctx, tid, keys, true)
		default:
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "Acción+invalida")
			return
		}

		if err != nil {
			log.Error("b2b: bulk action", "error", err, "action", action)
			b2bRedirectBack(w, r, "/admin/b2b/negocios", "error", "No+se+pudo+aplicar+la+acción")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/negocios", "success", strconv.Itoa(len(keys))+"+negocios+actualizados")
	}
}
