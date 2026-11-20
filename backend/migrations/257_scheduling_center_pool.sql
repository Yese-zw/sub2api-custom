-- 257: Scheduling center pools (调度中心：调度池 / 账号池)
-- 调度中心是独立的管理视图，不参与现有网关调度。
-- 分组下的账号默认都在「账号池」；只有被移入「调度池」的账号才在本表留一行，
-- 删除行即回到账号池。分组或账号删除时级联清理。
CREATE TABLE IF NOT EXISTS scheduling_center_pool_members (
    group_id   BIGINT      NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    account_id BIGINT      NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (group_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_scheduling_center_pool_members_account
    ON scheduling_center_pool_members (account_id);
