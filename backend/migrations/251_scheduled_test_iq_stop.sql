-- 251: Scheduled IQ check can pause account scheduling when the check fails
ALTER TABLE scheduled_test_plans ADD COLUMN IF NOT EXISTS iq_stop_on_fail BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE scheduled_test_plans ADD COLUMN IF NOT EXISTS iq_stop_minutes INT NOT NULL DEFAULT 60;
