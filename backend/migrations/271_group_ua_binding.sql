-- 271: 分组 User-Agent 绑定检测账号：开启后，命中规则的（黑名单）User-Agent 只能调度到规则指定的检测账号，
--      未命中任何规则的正常请求排除这些检测账号。
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS ua_binding_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS ua_binding_rules JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN groups.ua_binding_enabled IS
    'Enable User-Agent account binding: requests whose User-Agent matches a rule may only be scheduled to that rule''s detection accounts; other requests exclude all detection accounts';
COMMENT ON COLUMN groups.ua_binding_rules IS
    'User-Agent binding rules: [{name, patterns: [ua pattern], account_ids: [account id]}]';
