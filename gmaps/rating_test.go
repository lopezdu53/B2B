package gmaps

import "testing"

func TestMatchesRating(t *testing.T) {
	t.Parallel()

	if !MatchesRating(4.2, 0, 0) {
		t.Fatal("no filter should accept any rating")
	}

	if !MatchesRating(4.0, 4, 5.01) {
		t.Fatal("4.0 should be in 4-5")
	}

	if !MatchesRating(5.0, 4, 5.01) {
		t.Fatal("5.0 should be in 4-5")
	}

	if MatchesRating(3.9, 4, 5.01) {
		t.Fatal("3.9 should not be in 4-5")
	}

	if MatchesRating(3.0, 2, 3) {
		t.Fatal("3.0 is exclusive upper bound of 2-3")
	}

	if !MatchesRating(2.9, 2, 3) {
		t.Fatal("2.9 should be in 2-3")
	}
}

func TestFilterEntriesByRating(t *testing.T) {
	t.Parallel()

	in := []*Entry{
		{Title: "A", ReviewRating: 2.4},
		{Title: "B", ReviewRating: 3.5},
		{Title: "C", ReviewRating: 4.8},
		nil,
	}

	got := FilterEntriesByRating(in, 0, 0)
	if len(got) != 4 {
		t.Fatalf("no filter kept %d", len(got))
	}

	got = FilterEntriesByRating(append([]*Entry(nil), in...), 4, 5.01)
	if len(got) != 1 || got[0].Title != "C" {
		t.Fatalf("4-5: %+v", got)
	}

	got = FilterEntriesByRating(append([]*Entry(nil), in...), 2, 3)
	if len(got) != 1 || got[0].Title != "A" {
		t.Fatalf("2-3: %+v", got)
	}
}
