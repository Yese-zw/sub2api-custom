-- 258: Scheduling center group settings, managed test plans, rule-based pool entry
-- 调度中心为独立视图：这里的设置只生成智商检测定时测试、维护调度池归类，
-- 不会暂停账号，也不参与网关现有调度。

-- 智商检测记录首个回答内容到达的耗时（毫秒），其他测试为 NULL。
ALTER TABLE scheduled_test_results ADD COLUMN IF NOT EXISTS first_token_ms BIGINT;

-- 调度池成员来源：manual = 手动移入；rule = 规则自动移入（不再满足规则时自动移出）。
ALTER TABLE scheduling_center_pool_members
    ADD COLUMN IF NOT EXISTS source VARCHAR(16) NOT NULL DEFAULT 'manual';

-- 每个分组的调度中心设置。
CREATE TABLE IF NOT EXISTS scheduling_center_group_settings (
    group_id           BIGINT        PRIMARY KEY REFERENCES groups(id) ON DELETE CASCADE,
    enabled            BOOLEAN       NOT NULL DEFAULT false,
    cron_expression    VARCHAR(100)  NOT NULL DEFAULT '*/30 * * * *',
    rule_enabled       BOOLEAN       NOT NULL DEFAULT false,
    min_cache_rate     DOUBLE PRECISION,
    require_iq_pass    BOOLEAN       NOT NULL DEFAULT false,
    max_first_token_ms BIGINT,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- 调度中心为账号创建的智商检测定时测试计划（每个账号最多一个）。
-- 计划被删除时映射随之删除，下一轮同步会重新创建。
CREATE TABLE IF NOT EXISTS scheduling_center_account_plans (
    account_id BIGINT      PRIMARY KEY REFERENCES accounts(id) ON DELETE CASCADE,
    plan_id    BIGINT      NOT NULL UNIQUE REFERENCES scheduled_test_plans(id) ON DELETE CASCADE,
    group_id   BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
