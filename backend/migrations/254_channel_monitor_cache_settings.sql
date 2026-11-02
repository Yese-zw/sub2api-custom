-- 254: Channel monitor: per-monitor switch to show cache hit rate on the status card
CREATE TABLE IF NOT EXISTS channel_monitor_cache_settings (
    monitor_id      BIGINT PRIMARY KEY REFERENCES channel_monitors(id) ON DELETE CASCADE,
    show_cache_rate BOOLEAN NOT NULL DEFAULT false,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
