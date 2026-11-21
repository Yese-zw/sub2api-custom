-- 268: Exit rules on real requests become "K of the last W" instead of "K consecutive".
-- 出池规则改为「最近 W 次中有 K 次」：exit_latency_window / exit_error_window 为观察次数，
-- 原 exit_latency_count / exit_error_count 为触发次数。存量数据窗口取原次数，保持原有「连续 N 次」语义。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS exit_latency_window INT NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS exit_error_window INT NOT NULL DEFAULT 3;

UPDATE scheduling_center_group_settings
SET exit_latency_window = GREATEST(exit_latency_count, 1),
    exit_error_window = GREATEST(COALESCE(exit_error_count, 3), 1);
