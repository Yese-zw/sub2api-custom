-- 261: Make every scheduling center rule parameter configurable per group.
ALTER TABLE scheduling_center_group_settings
    -- 首 Token 取最近 N 次智商检测的平均值（无检测数据时用 24h 真实请求平均）
    ADD COLUMN IF NOT EXISTS first_token_test_samples  INT    NOT NULL DEFAULT 10,
    -- 入池：缓存率至少需要的 24h 请求数
    ADD COLUMN IF NOT EXISTS entry_cache_min_requests  INT    NOT NULL DEFAULT 1,
    -- 入池：要求最近 N 次智商检测都通过
    ADD COLUMN IF NOT EXISTS entry_iq_pass_count       INT    NOT NULL DEFAULT 1,
    -- 出池：真实请求只看入池之后且最近 N 小时
    ADD COLUMN IF NOT EXISTS exit_traffic_window_hours INT    NOT NULL DEFAULT 24,
    -- 退避：统计「坏」次数的时间窗口（分钟）
    ADD COLUMN IF NOT EXISTS bad_window_minutes        INT    NOT NULL DEFAULT 180,
    -- 退避：第 1、2、3… 次坏的冷却分钟数，最后一个为封顶值
    ADD COLUMN IF NOT EXISTS cooldown_minutes          INT[]  NOT NULL DEFAULT '{5,15,30,60}',
    -- 恢复：主动探测与实测的首 Token 上限（毫秒），NULL 表示不检查首 Token
    ADD COLUMN IF NOT EXISTS recover_latency_ms        BIGINT          DEFAULT 10000;
