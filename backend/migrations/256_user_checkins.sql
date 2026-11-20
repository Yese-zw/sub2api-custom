-- 256: Daily check-in (每日签到)
-- One row per user per calendar day (system timezone). The unique key is the
-- single source of truth for "already checked in today": concurrent requests
-- race on it and only one wins. Rewards are credited to users.balance in the
-- same transaction and mirrored into redeem_codes (type = 'checkin') so they
-- appear in the existing balance history.
CREATE TABLE IF NOT EXISTS user_checkins (
    id            BIGSERIAL PRIMARY KEY,
    user_id       BIGINT        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    checkin_date  DATE          NOT NULL,
    base_reward   DECIMAL(20,8) NOT NULL DEFAULT 0,
    streak_bonus  DECIMAL(20,8) NOT NULL DEFAULT 0,
    reward_amount DECIMAL(20,8) NOT NULL DEFAULT 0,
    streak_days   INTEGER       NOT NULL DEFAULT 1,
    ip_address    VARCHAR(64)   NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_user_checkins_user_date
    ON user_checkins (user_id, checkin_date);

CREATE INDEX IF NOT EXISTS idx_user_checkins_date
    ON user_checkins (checkin_date);
