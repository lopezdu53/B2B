package admin

import "testing"

func TestSearchDepthForSize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		n    int
		want int
	}{
		{100, 5},
		{500, 25},
		{1000, 50},
		{0, 25},
		{-1, 25},
	}
	for _, tc := range cases {
		if got := SearchDepthForSize(tc.n); got != tc.want {
			t.Errorf("SearchDepthForSize(%d)=%d want %d", tc.n, got, tc.want)
		}
	}
}
