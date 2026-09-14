package scraper

import (
	"context"
	"sync"
	"time"

	"github.com/gosom/google-maps-scraper/log"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	progressMinInterval = time.Second
	progressEveryN      = 5
)

// WriteJobProgress upserts a live extract count. Writes are throttled so a
// busy scrape does not hit the database on every listing.
func WriteJobProgress(db *pgxpool.Pool) func(riverJobID int64, count int) {
	if db == nil {
		return func(int64, int) {}
	}

	var (
		mu        sync.Mutex
		lastAt    time.Time
		lastID    int64
		lastCount int
	)

	return func(riverJobID int64, count int) {
		if riverJobID == 0 || count < 0 {
			return
		}

		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		sameJob := riverJobID == lastID

		if sameJob && !shouldWriteProgress(count, lastCount, now.Sub(lastAt)) {
			return
		}

		lastAt = now
		lastID = riverJobID
		lastCount = count

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		_, err := db.Exec(ctx, `
INSERT INTO scrape_job_progress (job_id, result_count, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (job_id) DO UPDATE SET
	result_count = EXCLUDED.result_count,
	updated_at = NOW()`, riverJobID, count)
		if err != nil {
			log.Debug("job progress write failed", "job_id", riverJobID, "error", err)
		}
	}
}

func shouldWriteProgress(count, lastCount int, since time.Duration) bool {
	if count <= 1 {
		return true
	}

	if count-lastCount >= progressEveryN {
		return true
	}

	return since >= progressMinInterval
}
