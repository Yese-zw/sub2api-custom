-- 252: Channel monitor IQ (降智) checks: per-monitor settings + rolling successful results
CREATE TABLE IF NOT EXISTS channel_monitor_iq_settings (
    monitor_id       BIGINT PRIMARY KEY REFERENCES channel_monitors(id) ON DELETE CASCADE,
    enabled          BOOLEAN NOT NULL DEFAULT false,
    creative_enabled BOOLEAN NOT NULL DEFAULT false,
    interval_minutes INT NOT NULL DEFAULT 10,
    models           JSONB NOT NULL DEFAULT '[]'::jsonb,
    last_run_at      TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS channel_monitor_iq_results (
    id          BIGSERIAL PRIMARY KEY,
    monitor_id  BIGINT NOT NULL REFERENCES channel_monitors(id) ON DELETE CASCADE,
    model       VARCHAR(200) NOT NULL,
    benchmark   VARCHAR(20) NOT NULL,
    verdict     VARCHAR(20) NOT NULL DEFAULT '',
    quality     VARCHAR(20) NOT NULL DEFAULT '',
    label       TEXT NOT NULL DEFAULT '',
    reason      TEXT NOT NULL DEFAULT '',
    answer      TEXT NOT NULL DEFAULT '',
    html        TEXT NOT NULL DEFAULT '',
    duration_ms BIGINT NOT NULL DEFAULT 0,
    started_at  TIMESTAMPTZ NOT NULL,
    finished_at TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_cmiq_results_lookup ON channel_monitor_iq_results(monitor_id, model, benchmark, created_at DESC);
