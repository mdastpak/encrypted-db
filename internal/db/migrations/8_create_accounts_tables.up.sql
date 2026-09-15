-- Migration: 8_create_accounts_tables.up.sql
-- Description: Create users, sub_accounts, api_keys, sessions tables

CREATE TYPE user_status AS ENUM ('ACTIVE', 'INACTIVE', 'SUSPENDED', 'BANNED', 'PENDING_VERIFICATION');
CREATE TYPE kyc_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'EXPIRED', 'UNDER_REVIEW');
CREATE TYPE kyc_tier AS ENUM ('NONE', 'BASIC', 'STANDARD', 'ENHANCED', 'INSTITUTIONAL');
CREATE TYPE sub_account_status AS ENUM ('ACTIVE', 'INACTIVE', 'FROZEN');
CREATE TYPE api_key_type AS ENUM ('HMAC', 'ED25519');
CREATE TYPE api_key_status AS ENUM ('ACTIVE', 'INACTIVE', 'REVOKED', 'EXPIRED');
CREATE TYPE auth_method AS ENUM ('PASSWORD', 'API_KEY', 'OAUTH', 'MAGIC_LINK', 'WEBAUTHN');
CREATE TYPE session_status AS ENUM ('ACTIVE', 'REVOKED', 'EXPIRED');

-- Users table (extends existing)
ALTER TABLE users ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone_verified BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret VARCHAR(256);
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS first_name VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_name VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS display_name VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url VARCHAR(500);
ALTER TABLE users ADD COLUMN IF NOT EXISTS status user_status NOT NULL DEFAULT 'PENDING_VERIFICATION';
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_at TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_login_ip INET;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referral_code VARCHAR(20) UNIQUE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS referred_by UUID REFERENCES users(id);
ALTER TABLE users ADD COLUMN IF NOT EXISTS kyc_status kyc_status NOT NULL DEFAULT 'PENDING';
ALTER TABLE users ADD COLUMN IF NOT EXISTS kyc_tier kyc_tier NOT NULL DEFAULT 'NONE';
ALTER TABLE users ADD COLUMN IF NOT EXISTS risk_score SMALLINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_referral_code ON users(referral_code);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);
CREATE INDEX IF NOT EXISTS idx_users_kyc_status ON users(kyc_status);

-- Sub-accounts
CREATE TABLE sub_accounts (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    label                       VARCHAR(100) NOT NULL,
    description                 TEXT,
    permissions                 JSONB NOT NULL DEFAULT '[]',
    daily_volume_limit          NUMERIC(36, 18),
    daily_withdrawal_limit      NUMERIC(36, 18),
    position_limit              NUMERIC(36, 18),
    status                      sub_account_status NOT NULL DEFAULT 'ACTIVE',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at                  TIMESTAMPTZ
);

CREATE INDEX idx_sub_accounts_user_id ON sub_accounts(user_id);
CREATE INDEX idx_sub_accounts_status ON sub_accounts(status);
CREATE INDEX idx_sub_accounts_deleted ON sub_accounts(deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TRIGGER update_sub_accounts_updated_at BEFORE UPDATE ON sub_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- API Keys
CREATE TABLE api_keys (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sub_account_id              UUID NOT NULL REFERENCES sub_accounts(id) ON DELETE CASCADE,
    name                        VARCHAR(100) NOT NULL,
    public_key                  VARCHAR(64) NOT NULL UNIQUE,
    key_type                    api_key_type NOT NULL DEFAULT 'HMAC',
    permissions                 JSONB NOT NULL DEFAULT '[]',
    ip_whitelist                TEXT[] DEFAULT '{}',
    rate_limit_rest             INTEGER NOT NULL DEFAULT 100,
    rate_limit_ws               INTEGER NOT NULL DEFAULT 50,
    rate_limit_trade            INTEGER NOT NULL DEFAULT 10,
    status                      api_key_status NOT NULL DEFAULT 'ACTIVE',
    last_used_at                TIMESTAMPTZ,
    last_used_ip                INET,
    expires_at                  TIMESTAMPTZ,
    secret_hash                 TEXT NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at                  TIMESTAMPTZ
);

CREATE INDEX idx_api_keys_sub_account_id ON api_keys(sub_account_id);
CREATE INDEX idx_api_keys_public_key ON api_keys(public_key);
CREATE INDEX idx_api_keys_status ON api_keys(status);

CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sessions
CREATE TABLE sessions (
    id                          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sub_account_id              UUID REFERENCES sub_accounts(id) ON DELETE SET NULL,
    auth_method                 VARCHAR(20) NOT NULL,
    access_token_hash           TEXT NOT NULL,
    refresh_token_hash          TEXT NOT NULL,
    user_agent                  TEXT,
    ip                          INET,
    device_id                   VARCHAR(64),
    status                      session_status NOT NULL DEFAULT 'ACTIVE',
    expires_at                  TIMESTAMPTZ NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at                  TIMESTAMPTZ
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_sub_account_id ON sessions(sub_account_id);
CREATE INDEX idx_sessions_refresh_token ON sessions(refresh_token_hash);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
CREATE INDEX idx_sessions_status ON sessions(status);

-- Trigger for updated_at on sub_accounts, api_keys
CREATE TRIGGER update_sub_accounts_updated_at BEFORE UPDATE ON sub_accounts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_api_keys_updated_at BEFORE UPDATE ON api_keys
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();