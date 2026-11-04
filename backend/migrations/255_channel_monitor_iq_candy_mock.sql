-- 255: Channel monitor IQ: candy test can use mock data (correct answer, no upstream request)
ALTER TABLE channel_monitor_iq_settings
    ADD COLUMN IF NOT EXISTS candy_mock BOOLEAN NOT NULL DEFAULT false;
