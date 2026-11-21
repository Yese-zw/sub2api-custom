-- 266: Configurable number of check records kept per account (scheduling center plans).
-- 每个账号保留最近 record_retention 条检测记录（同步到调度中心托管计划的 max_results）。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS record_retention INT NOT NULL DEFAULT 100;
