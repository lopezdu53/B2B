//nolint:testpackage // tests unexported dismiss helpers
package admin

import (
	"testing"

	"github.com/gosom/google-maps-scraper/rqueue"
)

func TestJobIsDismissed(t *testing.T) {
	if jobIsDismissed("anything", nil) {
		t.Fatal("empty set should not dismiss")
	}

	if jobIsDismissed("not-a-hash", map[int64]struct{}{1: {}}) {
		t.Fatal("undecodable id should stay visible")
	}
}

func TestDecodeJobIDRoundTripUsedByDismiss(t *testing.T) {
	// The dismiss route receives the same hashed id ListJobs returns.
	// An invalid hash must be rejected by the handler (400).
	if _, err := rqueue.DecodeJobID("nope"); err == nil {
		t.Fatal("expected invalid job id")
	}
}
