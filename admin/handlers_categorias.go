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

		cats, err := appState.Store.ListCategories(r.Context(), tid)
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

// CreateCategoryHandler creates a business category.
func CreateCategoryHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if SessionFromContext(r.Context()) == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		name := strings.TrimSpace(r.FormValue("name"))
		if name == "" {
			b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "El+nombre+es+obligatorio")
			return
		}

		tid, _ := effectiveTenant(appState, r)

		if _, err := appState.Store.CreateCategory(r.Context(), tid, name); err != nil {
			log.Error("b2b: create category", "error", err)
			b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "No+se+pudo+crear+(¿ya+existe?)")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/categorias", "success", "Categoría+creada")
	}
}

// DeleteCategoryHandler removes a category.
func DeleteCategoryHandler(appState *AppState) http.HandlerFunc {
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

		if err := appState.Store.DeleteCategory(r.Context(), tid, id); err != nil {
			log.Error("b2b: delete category", "error", err, "id", id)
			b2bRedirectBack(w, r, "/admin/b2b/categorias", "error", "No+se+pudo+eliminar")

			return
		}

		b2bRedirectBack(w, r, "/admin/b2b/categorias", "success", "Categoría+eliminada")
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

		tid, _ := effectiveTenant(appState, r)
		ctx := r.Context()

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
