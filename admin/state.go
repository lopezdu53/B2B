package admin

import (
	"embed"
	"fmt"
	"hash/crc32"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gosom/google-maps-scraper/ratelimit"
)

//go:embed templates/*.html
var templatesFS embed.FS

//go:embed static
var staticFS embed.FS

const DefaultCookieName = "gms_session"

// assetVersion is a short content hash of styles.css, appended to the
// stylesheet URL so browsers refetch it whenever it changes (cache busting).
var assetVersion = computeAssetVersion()

func computeAssetVersion() string {
	b, err := staticFS.ReadFile("static/styles.css")
	if err != nil {
		return "1"
	}

	return fmt.Sprintf("%08x", crc32.ChecksumIEEE(b))
}

// NewAppState creates a new AppState with all dependencies initialized.
func NewAppState(store IStore, rateLimiter ratelimit.Store, encryptionKey []byte) (*AppState, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}

	return &AppState{
		Store:         store,
		RateLimiter:   rateLimiter,
		Templates:     tmpl,
		EncryptionKey: encryptionKey,
		CookieName:    DefaultCookieName,
	}, nil
}

// StaticFileHandler returns an http.Handler for serving static files. Assets
// are versioned via a ?v= query on the stylesheet, so we allow long caching but
// still revalidate to pick up changes on redeploy.
func StaticFileHandler() http.Handler {
	staticContent, _ := fs.Sub(staticFS, "static")
	fileServer := http.StripPrefix("/admin/static/", http.FileServer(http.FS(staticContent)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		fileServer.ServeHTTP(w, r)
	})
}
