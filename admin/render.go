package admin

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/gosom/google-maps-scraper/cryptoext"
)

// resolveGoogleMapsAPIKey prefers the process env / serve flag, then a stored
// admin setting. The JS key is public (restrict it by HTTP referrer in GCP).
func resolveGoogleMapsAPIKey(appState *AppState, r *http.Request) string {
	if appState != nil {
		if k := strings.TrimSpace(appState.GoogleMapsAPIKey); k != "" {
			return k
		}

		if appState.Store != nil && r != nil {
			if cfg, err := appState.Store.GetConfig(r.Context(), ConfigGoogleMapsAPIKey); err == nil && cfg != nil {
				return strings.TrimSpace(cfg.Value)
			}
		}
	}

	return ""
}

// asJSON encodes v for safe embedding inside a <script> tag. encoding/json
// already escapes <, > and & so the payload cannot break out of the script.
func asJSON(v any) template.JS {
	b, err := json.Marshal(v)
	if err != nil {
		return template.JS("null")
	}

	return template.JS(b)
}

// renderTemplate renders a template with the given data.
func renderTemplate(appState *AppState, w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	if data == nil {
		data = make(map[string]any)
	}

	data["CSRFToken"] = CSRFTokenFromContext(r.Context())
	data["AssetVersion"] = assetVersion
	data["GoogleMapsAPIKey"] = resolveGoogleMapsAPIKey(appState, r)

	// Inject identity + active-tenant context for the sidebar.
	if user := UserFromContext(r.Context()); user != nil {
		data["CurrentUser"] = user
		data["IsSuperadmin"] = user.IsSuperadmin()
		data["IsAdvisor"] = user.IsAdvisor()
		data["CanManage"] = user.IsAdmin()

		tid, tenant := effectiveTenant(appState, r)
		data["ActiveTenantID"] = tid
		data["ActiveTenant"] = tenant

		if user.IsSuperadmin() {
			if tenants, err := appState.Store.ListTenants(r.Context()); err == nil {
				data["NavTenants"] = tenants
			}
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := appState.Templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// validatePassword validates a user's password.
func validatePassword(user *User, password string) error {
	if !cryptoext.VerifyPassword(password, user.PasswordHash) {
		return ErrInvalidPassword
	}

	return nil
}

// isSecureRequest checks if the request was made over HTTPS.
// Checks both direct TLS and X-Forwarded-Proto header for reverse proxy setups.
func isSecureRequest(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	// Check X-Forwarded-Proto header for reverse proxy setups
	if proto := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]); strings.EqualFold(proto, "https") {
		return true
	}

	return false
}

// splitBackupCodes splits a comma-separated string of backup codes.
func splitBackupCodes(codes string) []string {
	return strings.Split(codes, ",")
}
