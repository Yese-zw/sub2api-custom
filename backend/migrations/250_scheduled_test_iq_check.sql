-- 250: Scheduled tests can also run the candy IQ (降智) check
ALTER TABLE scheduled_test_plans ADD COLUMN IF NOT EXISTS iq_check BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE scheduled_test_results ADD COLUMN IF NOT EXISTS iq_status VARCHAR(20) NOT NULL DEFAULT '';
ALTER TABLE scheduled_test_results ADD COLUMN IF NOT EXISTS iq_message TEXT NOT NULL DEFAULT '';
