package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/gosom/google-maps-scraper/httpext"
	"github.com/gosom/google-maps-scraper/log"
)

// Routes sets up all admin routes.
func Routes(r chi.Router, appState *AppState, riverUIHandler http.Handler) {
	log.Debug("setting up admin routes")

	// Serve static files (CSS, JS, etc.) - no CSRF needed
	r.Handle("/admin/static/*", StaticFileHandler())

	r.Route("/admin", func(r chi.Router) {
		r.Use(httpext.LoggingMiddleware)
		r.Use(CSRFProtection(appState.EncryptionKey, appState.CookieName))

		r.Get("/login", LoginPageHandler(appState))
		r.Post("/login", LoginSubmitHandler(appState))

		r.Get("/login/2fa", TwoFactorVerifyPageHandler(appState))
		r.Post("/login/2fa", TwoFactorVerifySubmitHandler(appState))

		r.Post("/logout", LogoutHandler(appState))

		r.Group(func(r chi.Router) {
			r.Use(SessionAuth(appState.Store, appState.CookieName))

			r.Get("/", DashboardHandler(appState))

			// Tenant switching is superadmin-only (handler also enforces this).
			r.With(RequireSuperadmin).Post("/switch-tenant", SwitchTenantHandler(appState))

			// Superadmin-only: manage client companies.
			r.Group(func(r chi.Router) {
				r.Use(RequireSuperadmin)
				r.Get("/clientes", ClientesPageHandler(appState))
				r.Post("/clientes", CreateTenantHandler(appState))
			})

			// B2B map dashboard (prospecting / CRM) — reads + advisor-safe writes.
			r.Get("/b2b", B2BPageHandler(appState))
			r.Get("/b2b/negocios", NegociosPageHandler(appState))
			r.Get("/b2b/negocios/export", NegociosExportHandler(appState))
			r.Post("/b2b/negocios/bulk", BulkBusinessHandler(appState))
			r.Get("/b2b/papelera", PapeleraPageHandler(appState))
			r.Post("/b2b/papelera/bulk", PapeleraBulkHandler(appState))
			r.Get("/b2b/jobs", B2BJobsHandler(appState))
			r.Get("/b2b/summary", B2BSummaryHandler(appState))
			r.Get("/b2b/businesses", B2BBusinessesHandler(appState))
			r.Post("/b2b/business", B2BSetStatusHandler(appState))
			r.Post("/b2b/business/update", UpdateBusinessFormHandler(appState))

			// Tenant-admin only: team/zone/category management and scrape launches.
			r.Group(func(r chi.Router) {
				r.Use(RequireTenantAdmin)
				r.Get("/b2b/asesores", AsesoresPageHandler(appState))
				r.Get("/b2b/zonas", ZonasPageHandler(appState))
				r.Get("/b2b/categorias", CategoriasPageHandler(appState))
				r.Post("/b2b/categorias", CreateCategoryHandler(appState))
				r.Post("/b2b/categorias/{id}/update", UpdateCategoryHandler(appState))
				r.Post("/b2b/categorias/{id}/delete", DeleteCategoryHandler(appState))
				r.Post("/b2b/search", B2BSearchHandler(appState))
				r.Post("/b2b/advisors", CreateAdvisorHandler(appState))
				r.Post("/b2b/advisors/{id}/update", UpdateAdvisorHandler(appState))
				r.Post("/b2b/advisors/{id}/delete", DeleteAdvisorHandler(appState))
				r.Post("/b2b/zones", CreateZoneHandler(appState))
				r.Post("/b2b/zones/{id}/update", UpdateZoneHandler(appState))
				r.Post("/b2b/zones/{id}/delete", DeleteZoneHandler(appState))
			})

			r.Get("/settings", SettingsPageHandler(appState))
			r.Post("/settings/password", ChangePasswordHandler(appState))
			r.With(RequireSuperadmin).Post("/settings/maps-key", SaveGoogleMapsKeyHandler(appState))
			r.Get("/2fa/prompt", TwoFactorPromptPageHandler(appState))
			r.Get("/2fa/setup", TwoFactorSetupPageHandler(appState))
			r.Post("/2fa/setup", TwoFactorSetupSubmitHandler(appState))
			r.Post("/2fa/disable", TwoFactorDisableHandler(appState))
			// Platform-only pages (scraping infrastructure): superadmin only.
			r.Group(func(r chi.Router) {
				r.Use(RequireSuperadmin)
				r.Get("/api-keys", APIKeysPageHandler(appState))
				r.Post("/api-keys", CreateAPIKeyHandler(appState))
				r.Post("/api-keys/revoke", RevokeAPIKeyHandler(appState))
				r.Get("/jobs", JobsPageHandler(appState))
				r.Get("/jobs/{job_id}/download", DownloadJobResultsHandler(appState))
				r.Post("/jobs/{job_id}/delete", DeleteJobHandler(appState))
				r.Post("/jobs/delete", BatchDeleteJobsHandler(appState))
				r.Post("/jobs/delete-filtered", DeleteAllFilteredJobsHandler(appState))
				r.Get("/workers", WorkersPageHandler(appState))
				r.Get("/workers/stream", WorkersStreamHandler(appState))
				r.Post("/workers", ProvisionWorkerHandler(appState))
				r.Post("/workers/settings", SaveProviderTokenHandler(appState))
				r.Get("/workers/ssh-key/private", DownloadSSHKeyHandler(appState, "private"))
				r.Get("/workers/ssh-key/public", DownloadSSHKeyHandler(appState, "public"))
				r.Post("/workers/{id}/delete", DeleteWorkerHandler(appState))
				r.Get("/workers/{id}/terminal", TerminalPageHandler(appState))
				r.Get("/workers/{id}/terminal/ws", TerminalWSHandler(appState))
			})
		})
	})

	// Mount River UI under /riverui/ (requires session auth)
	r.Group(func(r chi.Router) {
		r.Use(SessionAuth(appState.Store, appState.CookieName))
		r.Mount("/riverui/", riverUIHandler)
	})
	log.Info("River UI available at /riverui/ (requires login)")

	// Public, read-only embeddable map (for PowerPoint's Web Viewer add-in and
	// similar). No session auth — access is gated by a per-tenant HMAC token.
	// Kept outside the /admin group so no CSRF/frame-blocking headers apply.
	r.Get("/embed/map", EmbedMapHandler(appState))
	r.Get("/embed/businesses", EmbedBusinessesHandler(appState))

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	// Root redirect
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
	})
}
