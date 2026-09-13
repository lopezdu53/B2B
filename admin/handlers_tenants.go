package admin

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gosom/google-maps-scraper/log"
)

// RequireTenantAdmin blocks advisor (and anonymous) users from tenant-management routes.
func RequireTenantAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil || !u.IsAdmin() {
			http.Error(w, "No autorizado", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// safeAdminPath returns dest if it is a same-app /admin path, otherwise fallback.
// Absolute URLs (e.g. Referer) keep only path+query when they point at /admin.
func safeAdminPath(raw, fallback string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}

	u, err := url.Parse(raw)
	if err != nil {
		return fallback
	}

	path := u.Path
	if u.RawQuery != "" {
		path += "?" + u.RawQuery
	}

	if path == "" || strings.HasPrefix(path, "//") || strings.ContainsAny(path, "\\\r\n") {
		return fallback
	}

	if path != "/admin" && !strings.HasPrefix(path, "/admin/") {
		return fallback
	}

	return path
}

// passwordMeetsPolicy reports whether pw satisfies the platform minimum.
func passwordMeetsPolicy(pw string) bool {
	return len(pw) >= MinPasswordLength
}

// effectiveTenant resolves which tenant's data the current request operates on:
//   - admin / advisor: their own tenant.
//   - superadmin: the "active" tenant chosen via cookie, else the first tenant.
//
// It returns the tenant id and (best-effort) the tenant record for display.
func effectiveTenant(appState *AppState, r *http.Request) (int64, *Tenant) {
	user := UserFromContext(r.Context())
	if user == nil {
		return 0, nil
	}

	if user.Role != RoleSuperadmin {
		if user.TenantID == nil {
			return 0, nil
		}

		t, _ := appState.Store.GetTenant(r.Context(), *user.TenantID)

		return *user.TenantID, t
	}

	// Superadmin: active tenant from cookie.
	if c, err := r.Cookie(activeTenantCookie); err == nil {
		if id, perr := strconv.ParseInt(c.Value, 10, 64); perr == nil {
			if t, gerr := appState.Store.GetTenant(r.Context(), id); gerr == nil {
				return id, t
			}
		}
	}

	// Fall back to the first tenant.
	tenants, _ := appState.Store.ListTenants(r.Context())
	if len(tenants) > 0 {
		return tenants[0].ID, &tenants[0]
	}

	return 1, nil
}

// advisorScope returns the advisor id a request's business data must be limited
// to. Advisor users see only their own assigned businesses; everyone else (admin,
// superadmin) sees the whole tenant, so it returns nil.
func advisorScope(r *http.Request) *int64 {
	u := UserFromContext(r.Context())
	if u != nil && u.Role == RoleAdvisor && u.AdvisorID != nil {
		return u.AdvisorID
	}

	return nil
}

// RequireSuperadmin blocks non-superadmin users from a route.
func RequireSuperadmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil || !u.IsSuperadmin() {
			http.Error(w, "No autorizado", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// tenantRow is a tenant enriched with its user/advisor counts for the list.
type tenantRow struct {
	Tenant

	Admins   int
	Advisors int
}

// ClientesPageHandler renders the superadmin's tenant (client) management page.
func ClientesPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenants, err := appState.Store.ListTenants(r.Context())
		if err != nil {
			log.Error("clientes: list tenants", "error", err)
		}

		activeID, _ := effectiveTenant(appState, r)

		rows := make([]tenantRow, 0, len(tenants))
		for i := range tenants {
			row := tenantRow{Tenant: tenants[i]}
			row.Admins, row.Advisors = appState.Store.TenantCounts(r.Context(), tenants[i].ID)
			rows = append(rows, row)
		}

		data := map[string]any{
			"Rows":     rows,
			"ActiveID": activeID,
			"Success":  r.URL.Query().Get("success"),
			"Error":    r.URL.Query().Get("error"),
		}

		renderTemplate(appState, w, r, "clientes.html", data)
	}
}

// CreateTenantHandler creates a client company plus its first admin user.
func CreateTenantHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimSpace(r.FormValue("name"))
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		if name == "" || username == "" || !passwordMeetsPolicy(password) {
			http.Redirect(w, r, "/admin/clientes?error=Nombre,+usuario+y+contraseña+(mín.+8+caracteres)+son+obligatorios", http.StatusSeeOther)
			return
		}

		tenant, err := appState.Store.CreateTenant(r.Context(), name)
		if err != nil {
			log.Error("clientes: create tenant", "error", err)
			http.Redirect(w, r, "/admin/clientes?error=No+se+pudo+crear+el+cliente", http.StatusSeeOther)

			return
		}

		// Seed the new client with the common categories (with a color + icon).
		for _, c := range []struct{ name, color, icon string }{
			{"Restaurantes", "#e11d48", "🍽️"},
			{"Hoteles", "#7c3aed", "🏨"},
			{"Supermercados", "#059669", "🛒"},
		} {
			_, _ = appState.Store.CreateCategory(r.Context(), tenant.ID, c.name, c.color, c.icon)
		}

		if _, err := appState.Store.CreateTenantUser(r.Context(), username, password, RoleAdmin, tenant.ID); err != nil {
			if err == ErrUserExists {
				http.Redirect(w, r, "/admin/clientes?error=Ese+usuario+ya+existe", http.StatusSeeOther)
				return
			}

			log.Error("clientes: create tenant admin", "error", err)
			http.Redirect(w, r, "/admin/clientes?error=Cliente+creado+pero+falló+el+usuario+admin", http.StatusSeeOther)

			return
		}

		http.Redirect(w, r, "/admin/clientes?success=Cliente+y+admin+creados", http.StatusSeeOther)
	}
}

// SwitchTenantHandler lets the superadmin choose which tenant to view.
func SwitchTenantHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil || !u.IsSuperadmin() {
			http.Error(w, "No autorizado", http.StatusForbidden)
			return
		}

		id := strings.TrimSpace(r.FormValue("tenant_id"))
		if _, err := strconv.ParseInt(id, 10, 64); err != nil {
			http.Redirect(w, r, "/admin/b2b", http.StatusSeeOther)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     activeTenantCookie,
			Value:    id,
			Path:     "/",
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
			Secure:   isSecureRequest(r),
		})

		http.Redirect(w, r, safeAdminPath(r.Referer(), "/admin/b2b"), http.StatusSeeOther)
	}
}
