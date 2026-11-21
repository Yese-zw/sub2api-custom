-- 264: Scheduling center fallback accounts (兜底账号).
-- 分组开启调度后，调度池账号数低于 fallback_min_accounts 时，从账号池按 fallback_order
-- 依次比较（rate_multiplier 低优先 / first_token 低优先 / cache_rate 高优先）挑选账号补入调度池；
-- 非兜底成员够数后，兜底账号自动退回账号池。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS fallback_enabled         BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN IF NOT EXISTS fallback_min_accounts    INT     NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS fallback_order           TEXT[]  NOT NULL DEFAULT '{rate_multiplier,first_token,cache_rate}',
    ADD COLUMN IF NOT EXISTS fallback_require_iq_pass BOOLEAN NOT NULL DEFAULT true;
