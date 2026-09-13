package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserRoles(t *testing.T) {
	t.Parallel()

	cases := []struct {
		role                  string
		admin, super, advisor bool
	}{
		{RoleSuperadmin, true, true, false},
		{RoleAdmin, true, false, false},
		{RoleAdvisor, false, false, true},
		{"", false, false, false},
	}

	for _, tc := range cases {
		u := &User{Role: tc.role}
		if got := u.IsAdmin(); got != tc.admin {
			t.Errorf("%s IsAdmin: got %v want %v", tc.role, got, tc.admin)
		}
		if got := u.IsSuperadmin(); got != tc.super {
			t.Errorf("%s IsSuperadmin: got %v want %v", tc.role, got, tc.super)
		}
		if got := u.IsAdvisor(); got != tc.advisor {
			t.Errorf("%s IsAdvisor: got %v want %v", tc.role, got, tc.advisor)
		}
	}

	var none *User
	if none.IsAdmin() || none.IsSuperadmin() || none.IsAdvisor() {
		t.Error("nil user must not match any role")
	}
}

func TestSafeAdminPath(t *testing.T) {
	t.Parallel()

	const fallback = "/admin/b2b"

	cases := map[string]string{
		"":                                    fallback,
		"/admin":                              "/admin",
		"/admin/b2b/negocios":                 "/admin/b2b/negocios",
		"/admin/b2b/negocios?q=foo":           "/admin/b2b/negocios?q=foo",
		"https://app.example/admin/clientes":  "/admin/clientes",
		"https://evil.example/admin/phishing": "/admin/phishing", // host stripped; stays on-app
		"https://evil.example/login":          fallback,
		"//evil.example/admin":                "/admin", // protocol-relative: only the path is kept
		"/not-admin":                          fallback,
		"https://evil.example/?next=/admin/":  fallback,
	}

	for raw, want := range cases {
		if got := safeAdminPath(raw, fallback); got != want {
			t.Errorf("safeAdminPath(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestPasswordMeetsPolicy(t *testing.T) {
	t.Parallel()

	if passwordMeetsPolicy("short") {
		t.Error("password shorter than MinPasswordLength accepted")
	}
	if !passwordMeetsPolicy(strings.Repeat("x", MinPasswordLength)) {
		t.Error("minimum-length password rejected")
	}
}

func TestUniqueKeys(t *testing.T) {
	t.Parallel()

	got := uniqueKeys([]string{" a ", "", "b", "a", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("uniqueKeys = %#v", got)
	}
}

func TestAsJSONEscapesScriptBreakout(t *testing.T) {
	t.Parallel()

	payload := asJSON(map[string]string{"title": "</script><img src=x onerror=alert(1)>"})
	s := string(payload)
	if strings.Contains(s, "</script>") {
		t.Fatalf("JSON still contains raw </script>: %s", s)
	}

	var decoded map[string]string
	if err := json.Unmarshal([]byte(s), &decoded); err != nil {
		t.Fatalf("asJSON is not valid JSON: %v (%s)", err, s)
	}
	if decoded["title"] != "</script><img src=x onerror=alert(1)>" {
		t.Fatalf("round-trip lost title: %q", decoded["title"])
	}
}

func TestRequireTenantAdminRejectsAdvisor(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := RequireTenantAdmin(inner)

	req := httptest.NewRequest(http.MethodGet, "/admin/b2b/asesores", nil)
	req = req.WithContext(withUser(req.Context(), &User{Role: RoleAdvisor}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("advisor status = %d, want 403", rec.Code)
	}
}

func TestRequireTenantAdminAllowsAdmin(t *testing.T) {
	t.Parallel()

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
	h := RequireTenantAdmin(inner)

	req := httptest.NewRequest(http.MethodGet, "/admin/b2b/asesores", nil)
	req = req.WithContext(withUser(req.Context(), &User{Role: RoleAdmin}))

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("admin status = %d, want 204", rec.Code)
	}
}

func TestAdvisorScope(t *testing.T) {
	t.Parallel()

	id := int64(9)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withUser(req.Context(), &User{Role: RoleAdvisor, AdvisorID: &id}))

	got := advisorScope(req)
	if got == nil || *got != id {
		t.Fatalf("advisorScope = %v, want %d", got, id)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/", nil)
	adminReq = adminReq.WithContext(withUser(adminReq.Context(), &User{Role: RoleAdmin}))
	if advisorScope(adminReq) != nil {
		t.Fatal("admin must not have advisor scope")
	}
}

func withUser(ctx context.Context, user *User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}
