-- 263: Scheduling center latency probe.
-- 调度中心的检测在智商题之外，再按「测试账号连接 → 常规请求（hi）」测一次速：
-- first_token_ms 记录测速的首 Token，probe_total_ms 记录测速的总耗时（latency_ms 仍是智商题耗时）。
ALTER TABLE scheduled_test_results ADD COLUMN IF NOT EXISTS probe_total_ms BIGINT;
