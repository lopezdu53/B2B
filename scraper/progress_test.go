package scraper

import (
	"testing"
	"time"

	"github.com/gosom/google-maps-scraper/gmaps"
	"github.com/stretchr/testify/require"
)

func TestShouldWriteProgress(t *testing.T) {
	if !shouldWriteProgress(1, 0, 0) {
		t.Fatal("first listing should write")
	}

	if shouldWriteProgress(3, 1, 200*time.Millisecond) {
		t.Fatal("small increment inside a second should wait")
	}

	if !shouldWriteProgress(6, 1, 200*time.Millisecond) {
		t.Fatal("every fifth extra listing should write")
	}

	if !shouldWriteProgress(2, 1, time.Second) {
		t.Fatal("a full second should write")
	}
}

func TestWriteJobProgressNilDB(t *testing.T) {
	fn := WriteJobProgress(nil)
	require.NotPanics(t, func() { fn(1, 3) })
}

func TestCentralWriter_OnProgressReportsCounts(t *testing.T) {
	cw := NewCentralWriter(nil, noopSave(nil))

	var got []int

	cw.OnProgress = func(riverJobID int64, count int) {
		require.Equal(t, int64(100), riverJobID)
		got = append(got, count)
	}

	cw.RegisterJob("job1", 100, "restaurants")
	cw.AddResult("job1", &gmaps.Entry{Title: "A"})
	cw.AddResult("job1", &gmaps.Entry{Title: "B"})
	cw.AddResult("other", &gmaps.Entry{Title: "Ignored"})

	require.Equal(t, []int{1, 2}, got)
}
