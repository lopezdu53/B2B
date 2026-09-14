-- +migrate Up

-- Live extract counts while a scrape is still running. scrape_results is
-- written only at flush; writing a stub row there would trip CRM ingest.
CREATE TABLE IF NOT EXISTS scrape_job_progress (
    job_id BIGINT PRIMARY KEY,
    result_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Hidden from the B2B "búsquedas recientes" list. scrape_results stay so
-- businesses already on the map are not removed.
CREATE TABLE IF NOT EXISTS b2b_dismissed_jobs (
    job_id BIGINT PRIMARY KEY,
    dismissed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +migrate Down
DROP TABLE IF EXISTS b2b_dismissed_jobs;
DROP TABLE IF EXISTS scrape_job_progress;
