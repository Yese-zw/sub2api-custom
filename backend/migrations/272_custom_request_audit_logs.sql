-- 网关请求审计：记录客户端发往网关的完整请求体与网关返回给客户端的完整响应体。
-- 由管理员在「使用记录 → 请求审计设置」开关；按保留天数自动清理，单个 body 按设置截断。
-- 与 usage_logs 通过 client_request_id / request_id 关联（usage_logs.request_id 形如 client:<id> / local:<id>）。
CREATE TABLE IF NOT EXISTS request_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    request_id VARCHAR(128) NOT NULL DEFAULT '',
    client_request_id VARCHAR(128) NOT NULL DEFAULT '',
    user_id BIGINT,
    api_key_id BIGINT,
    account_id BIGINT,
    method VARCHAR(16) NOT NULL DEFAULT '',
    path VARCHAR(512) NOT NULL DEFAULT '',
    model VARCHAR(255) NOT NULL DEFAULT '',
    stream BOOLEAN NOT NULL DEFAULT FALSE,
    status_code INT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    client_ip VARCHAR(64) NOT NULL DEFAULT '',
    user_agent VARCHAR(512) NOT NULL DEFAULT '',
    request_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_content_type VARCHAR(255) NOT NULL DEFAULT '',
    request_body TEXT NOT NULL DEFAULT '',
    request_body_size BIGINT NOT NULL DEFAULT 0,
    request_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    response_content_type VARCHAR(255) NOT NULL DEFAULT '',
    response_body TEXT NOT NULL DEFAULT '',
    response_body_size BIGINT NOT NULL DEFAULT 0,
    response_truncated BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_request_audit_logs_created_at
    ON request_audit_logs (created_at);
CREATE INDEX IF NOT EXISTS idx_request_audit_logs_client_request_id
    ON request_audit_logs (client_request_id) WHERE client_request_id <> '';
CREATE INDEX IF NOT EXISTS idx_request_audit_logs_request_id
    ON request_audit_logs (request_id) WHERE request_id <> '';
CREATE INDEX IF NOT EXISTS idx_request_audit_logs_api_key_created
    ON request_audit_logs (api_key_id, created_at);
