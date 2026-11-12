-- 系统可用余额：全站一笔成本账，和用户余额无关。
-- 新写入的 usage_logs 按账号计费扣减；删除或修改使用记录不再回写这笔账。
-- 历史使用记录不回填。管理员只能充值或扣除，不能覆盖当前值。

CREATE TABLE IF NOT EXISTS system_available_balance (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    balance DECIMAL(20, 10) NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO system_available_balance (id, balance)
VALUES (1, 0)
ON CONFLICT (id) DO NOTHING;

CREATE OR REPLACE FUNCTION deduct_system_available_balance()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    cost DECIMAL(20, 10);
BEGIN
    -- 与管理端「账号计费」一致：(account_stats_cost ?? total_cost) * (account_rate_multiplier ?? 1)
    cost := COALESCE(NEW.account_stats_cost, NEW.total_cost, 0) * COALESCE(NEW.account_rate_multiplier, 1);
    IF cost = 0 THEN
        RETURN NEW;
    END IF;

    UPDATE system_available_balance
    SET balance = balance - cost,
        updated_at = NOW()
    WHERE id = 1;

    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS usage_logs_deduct_system_available_balance ON usage_logs;
CREATE TRIGGER usage_logs_deduct_system_available_balance
AFTER INSERT ON usage_logs
FOR EACH ROW
EXECUTE FUNCTION deduct_system_available_balance();
