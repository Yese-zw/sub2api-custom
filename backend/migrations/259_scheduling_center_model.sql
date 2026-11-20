-- 259: Scheduling center group settings choose the IQ check model.
-- 空字符串表示使用各平台默认测试模型（与定时测试计划 model_id 为空时一致）。
ALTER TABLE scheduling_center_group_settings
    ADD COLUMN IF NOT EXISTS model_id VARCHAR(100) NOT NULL DEFAULT '';
