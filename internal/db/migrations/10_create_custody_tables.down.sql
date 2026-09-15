-- Migration: 10_create_custody_tables.down.sql
-- Description: Drop custody tables

DROP TRIGGER IF EXISTS update_withdrawals_updated_at ON withdrawals;
DROP TRIGGER IF EXISTS update_custody_configs_updated_at ON custody_configs;

DROP TABLE IF EXISTS custody_configs;
DROP TABLE IF EXISTS settlements;
DROP TABLE IF EXISTS withdrawals;
DROP TABLE IF EXISTS deposits;
DROP TABLE IF EXISTS deposit_addresses;

DROP TYPE IF EXISTS custody_provider_type;
DROP TYPE IF EXISTS deposit_status_type;
DROP TYPE IF EXISTS withdrawal_status_type;
DROP TYPE IF EXISTS fee_level;
DROP TYPE IF EXISTS settlement_type;
DROP TYPE IF EXISTS settlement_status;