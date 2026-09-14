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

func TestFillCategoryNameInfersSpecialty(t *testing.T) {
	t.Parallel()

	row := businessRow{MapBusiness: MapBusiness{Specialty: "Bed & breakfast"}}
	fillCategoryName(&row, nil)
	if row.CategoryName != "Hoteles" {
		t.Fatalf("got %q", row.CategoryName)
	}

	hotelesID := int64(3)
	row = businessRow{MapBusiness: MapBusiness{CategoryID: &hotelesID, Specialty: "Hostales"}}
	fillCategoryName(&row, map[int64]string{3: "Hoteles"})
	if row.CategoryName != "Hoteles" {
		t.Fatalf("crm category: %q", row.CategoryName)
	}
}

func TestRatingBandBounds(t *testing.T) {
	t.Parallel()

	min, max, band, ok := RatingBandBounds("2-3")
	if !ok || min != 2 || max != 3 || band != RatingBand23 {
		t.Fatalf("2-3: %v %v %q %v", min, max, band, ok)
	}

	min, max, band, ok = RatingBandBounds("4-5")
	if !ok || min != 4 || max != 5.01 || band != RatingBand45 {
		t.Fatalf("4-5: %v %v %q %v", min, max, band, ok)
	}

	if _, _, _, ok = RatingBandBounds(""); ok {
		t.Fatal("empty band should be off")
	}

	var f BusinessFilter
	f.SetRatingBand("3-4")
	if f.RatingMin != 3 || f.RatingMaxExcl != 4 || f.RatingBand != RatingBand34 {
		t.Fatalf("set 3-4: %+v", f)
	}

	f.SetRatingBand("nope")
	if f.RatingMin != 0 || f.RatingMaxExcl != 0 || f.RatingBand != "" {
		t.Fatalf("invalid should clear: %+v", f)
	}
}
