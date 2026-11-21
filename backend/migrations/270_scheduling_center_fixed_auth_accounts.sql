-- 270: Scheduling center only manages Key accounts; OAuth / Setup Token accounts are always in the scheduling pool.
-- 调度中心只处理 Key 类型账号：OAuth / Setup Token 账号固定在调度池（查询时虚拟加入，不落库），
-- 清理它们已有的调度池记录与调度状态，避免残留的冷却 / 恢复状态继续参与评估。
DELETE FROM scheduling_center_pool_members m
USING accounts a
WHERE a.id = m.account_id AND a.type IN ('oauth', 'setup-token');

DELETE FROM scheduling_center_account_states s
USING accounts a
WHERE a.id = s.account_id AND a.type IN ('oauth', 'setup-token');
