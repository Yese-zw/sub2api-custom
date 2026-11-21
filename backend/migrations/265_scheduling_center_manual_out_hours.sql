-- 265: Manually removed accounts become eligible for automatic entry again after N hours.
-- 手动移出的账号在 manual_out_hours 小时后恢复为普通账号池状态（入池规则 / 兜底可再次选中）；0 表示永久。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS manual_out_hours INT NOT NULL DEFAULT 24;
