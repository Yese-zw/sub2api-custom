-- 260: Scheduling center exit rules, account lifecycle state and scheduling log.
-- 调度中心为独立视图：这里的状态只描述调度中心自己的调度池，与账号管理中的调度状态无关。

-- 出池规则（每个分组单独设置）。阈值列为 NULL 表示该条件关闭。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS exit_enabled            BOOLEAN          NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS exit_cache_drop_points  DOUBLE PRECISION          DEFAULT 20,
    ADD COLUMN IF NOT EXISTS exit_cache_min_requests INT              NOT NULL DEFAULT 20,
    ADD COLUMN IF NOT EXISTS exit_iq_window          INT              NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS exit_iq_max_degraded    INT                       DEFAULT 3,
    ADD COLUMN IF NOT EXISTS exit_latency_ms         BIGINT                    DEFAULT 10000,
    ADD COLUMN IF NOT EXISTS exit_latency_count      INT              NOT NULL DEFAULT 3,
    ADD COLUMN IF NOT EXISTS exit_error_count        INT                       DEFAULT 3,
    ADD COLUMN IF NOT EXISTS recover_success_count   INT              NOT NULL DEFAULT 3;

-- 不在调度池中的账号的调度中心状态（在调度池中的账号以 pool_members 为准，不在此表）。
--   cooling    出池后冷却中，cooldown_until 到期后进入 probing
--   probing    恢复检测：先主动探测（probe_status running → passed），再看真实请求连续正常
--   ready      恢复完成，等待入池规则
--   manual_out 手动移出，入池规则不会自动加回
CREATE TABLE IF NOT EXISTS scheduling_center_account_states (
    group_id         BIGINT      NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    account_id       BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    state            VARCHAR(16) NOT NULL,
    prev_source      VARCHAR(16) NOT NULL DEFAULT '',
    reason_code      VARCHAR(32) NOT NULL DEFAULT '',
    cooldown_until   TIMESTAMPTZ,
    probe_status     VARCHAR(16) NOT NULL DEFAULT '',
    probe_started_at TIMESTAMPTZ,
    probe_passed_at  TIMESTAMPTZ,
    recover_ok_count INT         NOT NULL DEFAULT 0,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, account_id)
);

-- 调度日志：账号入池、出池、冷却结束、探测、实测等每一步及原因。
-- is_bad 标记一次「坏」（规则出池 / 探测失败 / 实测失败），用于 3 小时内的退避计数。
CREATE TABLE IF NOT EXISTS scheduling_center_pool_events (
    id         BIGSERIAL   PRIMARY KEY,
    group_id   BIGINT      NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    account_id BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    action     VARCHAR(16) NOT NULL,
    source     VARCHAR(16) NOT NULL DEFAULT '',
    reason     VARCHAR(32) NOT NULL DEFAULT '',
    detail     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    is_bad     BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_scheduling_center_events_group_created
    ON scheduling_center_pool_events (group_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scheduling_center_events_bad
    ON scheduling_center_pool_events (group_id, account_id, created_at DESC) WHERE is_bad;
