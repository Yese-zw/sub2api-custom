-- 269: Per-group switch that makes the gateway schedule only from the scheduling center pool.
-- 调度池生效（灰度开关）：分组开启调度且 enforce_enabled 时，网关只从该分组调度池（含恢复观察中的账号）选账号；
-- 调度池无可用账号或池内账号全部选不上时回退到原有逻辑。默认关闭，保持影子模式。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS enforce_enabled BOOLEAN NOT NULL DEFAULT FALSE;
