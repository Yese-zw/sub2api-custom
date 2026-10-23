-- Per-user desktop App group.
-- app_group_id: 用户在个人资料中为桌面 App 选择的分组（0 = 跟随系统设置中的 App 默认分组）。
-- 由 /api/v1/user/app-group 读写；App 用它在对应分组下创建/使用 API Key。
ALTER TABLE users ADD COLUMN IF NOT EXISTS app_group_id bigint NOT NULL DEFAULT 0;

COMMENT ON COLUMN users.app_group_id IS '桌面 App 使用的分组；0 表示跟随系统 App 默认分组。';
