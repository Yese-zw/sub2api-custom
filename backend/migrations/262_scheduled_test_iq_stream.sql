-- 262: Only scheduling center plans run the IQ check as a streaming request.
-- 流式请求用于记录首 Token；其他定时智商检测保持原来的非流式请求，行为不变。
ALTER TABLE scheduled_test_plans ADD COLUMN IF NOT EXISTS iq_stream BOOLEAN NOT NULL DEFAULT false;

-- 已由调度中心创建的计划改为流式。
UPDATE scheduled_test_plans p
SET iq_stream = true
FROM scheduling_center_account_plans m
WHERE m.plan_id = p.id AND p.iq_stream = false;
