-- 253: Channel monitor IQ: choose which tests to run (candy / pelican); creative is a pelican variant
ALTER TABLE channel_monitor_iq_settings
    ADD COLUMN IF NOT EXISTS candy_enabled   BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS pelican_enabled BOOLEAN NOT NULL DEFAULT true;
