package admin

import (
	"net/http"
	"strings"

	"github.com/gosom/google-maps-scraper/cryptoext"
	"github.com/gosom/google-maps-scraper/log"
)

// SettingsPageHandler renders the settings page.
func SettingsPageHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := SessionFromContext(r.Context())
		if session == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		user, err := appState.Store.GetUserByID(r.Context(), session.UserID)
		if err != nil {
			http.Redirect(w, r, "/admin/login?error=User+not+found", http.StatusSeeOther)
			return
		}

		key := resolveGoogleMapsAPIKey(appState, r)
		masked := ""
		if n := len(key); n > 8 {
			masked = key[:4] + "…" + key[n-4:]
		} else if n > 0 {
			masked = "configurada"
		}

		data := map[string]any{
			"Username":          user.Username,
			"TOTPEnabled":       user.TOTPEnabled,
			"GoogleMapsKeySet":  key != "",
			"GoogleMapsKeyMask": masked,
			"Success":           r.URL.Query().Get("success"),
			"Error":             r.URL.Query().Get("error"),
		}
		renderTemplate(appState, w, r, "settings.html", data)
	}
}

// ChangePasswordHandler handles password change requests.
func ChangePasswordHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := SessionFromContext(r.Context())
		if session == nil {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}

		user, err := appState.Store.GetUserByID(r.Context(), session.UserID)
		if err != nil {
			http.Redirect(w, r, "/admin/settings?error=User+not+found", http.StatusSeeOther)
			return
		}

		currentPassword := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_password")

		// Validate current password
		if err := validatePassword(user, currentPassword); err != nil {
			http.Redirect(w, r, "/admin/settings?error=Current+password+is+incorrect", http.StatusSeeOther)
			return
		}

		// Validate new password
		if len(newPassword) < MinPasswordLength {
			http.Redirect(w, r, "/admin/settings?error=New+password+must+be+at+least+8+characters", http.StatusSeeOther)
			return
		}

		if newPassword != confirmPassword {
			http.Redirect(w, r, "/admin/settings?error=New+passwords+do+not+match", http.StatusSeeOther)
			return
		}

		// Check that new password is different from current
		if cryptoext.VerifyPassword(newPassword, user.PasswordHash) {
			http.Redirect(w, r, "/admin/settings?error=New+password+must+be+different+from+current+password", http.StatusSeeOther)
			return
		}

		// Update password
		if err := appState.Store.UpdatePassword(r.Context(), user.Username, newPassword); err != nil {
			http.Redirect(w, r, "/admin/settings?error=Failed+to+update+password", http.StatusSeeOther)
			return
		}

		// Invalidate all other sessions for this user
		if err := appState.Store.DeleteUserSessionsExcept(r.Context(), session.UserID, session.ID); err != nil {
			log.Error("failed to invalidate sessions after password change", "error", err, "user_id", session.UserID)
		}

		log.Info("audit", "action", "password_change", "user_id", session.UserID, "ip", r.RemoteAddr)

		http.Redirect(w, r, "/admin/settings?success=Password+updated+successfully", http.StatusSeeOther)
	}
}

// SaveGoogleMapsKeyHandler stores the Maps JavaScript API key (superadmin).
func SaveGoogleMapsKeyHandler(appState *AppState) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil || !u.IsSuperadmin() {
			http.Error(w, "No autorizado", http.StatusForbidden)
			return
		}

		key := strings.TrimSpace(r.FormValue("google_maps_api_key"))
		if key == "" {
			_ = appState.Store.DeleteConfig(r.Context(), ConfigGoogleMapsAPIKey)
			http.Redirect(w, r, "/admin/settings?success=Clave+de+Google+Maps+eliminada", http.StatusSeeOther)
			return
		}

		if err := appState.Store.SetConfig(r.Context(), &AppConfig{
			Key:   ConfigGoogleMapsAPIKey,
			Value: key,
		}, true); err != nil {
			log.Error("settings: save google maps key", "error", err)
			http.Redirect(w, r, "/admin/settings?error=No+se+pudo+guardar+la+clave", http.StatusSeeOther)
			return
		}

		http.Redirect(w, r, "/admin/settings?success=Clave+de+Google+Maps+guardada", http.StatusSeeOther)
	}
}
