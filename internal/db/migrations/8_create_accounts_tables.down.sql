-- Migration: 8_create_accounts_tables.down.sql
-- Description: Drop accounts tables

DROP TRIGGER IF EXISTS update_sub_accounts_updated_at ON sub_accounts;
DROP TRIGGER IF EXISTS update_api_keys_updated_at ON api_keys;

DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS sub_accounts;

DROP TYPE IF EXISTS user_status;
DROP TYPE IF EXISTS kyc_status;
DROP TYPE IF EXISTS kyc_tier;
DROP TYPE IF EXISTS sub_account_status;
DROP TYPE IF EXISTS api_key_type;
DROP TYPE IF EXISTS api_key_status;
DROP TYPE IF EXISTS auth_method;
DROP TYPE IF EXISTS session_status;