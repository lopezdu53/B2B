package admin

import "testing"

func TestSearchDepthForSize(t *testing.T) {
	t.Parallel()

	cases := []struct {
		n    int
		want int
	}{
		{50, 3},
		{100, 5},
		{250, 13},
		{1000, 15},
		{0, 5},
		{-1, 5},
	}
	for _, tc := range cases {
		if got := SearchDepthForSize(tc.n); got != tc.want {
			t.Errorf("SearchDepthForSize(%d)=%d want %d", tc.n, got, tc.want)
		}
	}
}
