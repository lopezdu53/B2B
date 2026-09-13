package admin

import "testing"

func TestValidBusinessStatusIncludesFeatured(t *testing.T) {
	t.Parallel()

	for _, st := range []string{StatusProspect, StatusInProgress, StatusClient, StatusFeatured, StatusDiscarded} {
		if !ValidBusinessStatus(st) {
			t.Errorf("%q should be valid", st)
		}
	}

	if ValidBusinessStatus("vip") {
		t.Fatal("unknown status should be invalid")
	}
}

func TestAbsoluteHTTPURL(t *testing.T) {
	t.Parallel()

	if got := AbsoluteHTTPURL("www.example.com"); got != "https://www.example.com" {
		t.Fatalf("got %q", got)
	}

	if got := AbsoluteHTTPURL("https://maps.google.com"); got != "https://maps.google.com" {
		t.Fatalf("got %q", got)
	}

	biz := MapBusiness{Website: "foo.com", Email: "a@b.com"}
	if biz.WebsiteURL() != "https://foo.com" {
		t.Fatalf("website %q", biz.WebsiteURL())
	}

	if biz.EmailURL() != "mailto:a@b.com" {
		t.Fatalf("email %q", biz.EmailURL())
	}
}
