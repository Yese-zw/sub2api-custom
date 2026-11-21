-- 267: Configurable retention (days) for scheduling center pool events.
-- 调度日志保留天数：后台定期删除超过 event_retention_days 天的日志（最少 8 天，覆盖 7 天的对比 / 导出窗口与「坏」次数窗口）。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS event_retention_days INT NOT NULL DEFAULT 30;
