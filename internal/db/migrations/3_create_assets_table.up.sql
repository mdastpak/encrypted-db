-- Migration: 3_create_assets_table.up.sql
-- Description: Create assets table for multi-asset support (fiat, crypto, tokens, stablecoins)

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE asset_type AS ENUM ('FIAT', 'COIN', 'TOKEN', 'STABLE', 'WRAPPED');
CREATE TYPE asset_status AS ENUM ('ACTIVE', 'INACTIVE', 'DEPRECATED', 'DELISTED');
CREATE TYPE settlement_mode AS ENUM ('OFF_CHAIN', 'ON_CHAIN');

CREATE TABLE assets (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    symbol          VARCHAR(20) NOT NULL UNIQUE,
    name            VARCHAR(100) NOT NULL,
    type            asset_type NOT NULL,
    status          asset_status NOT NULL DEFAULT 'INACTIVE',
    decimals        SMALLINT NOT NULL DEFAULT 8,
    
    -- Blockchain info
    network         VARCHAR(50),
    contract_address VARCHAR(100),
    native_symbol   VARCHAR(20),
    
    -- Display
    description     TEXT,
    logo_url        VARCHAR(500),
    website         VARCHAR(500),
    explorer_url    VARCHAR(500),
    
    -- Trading config
    min_withdrawal  NUMERIC(36, 18) NOT NULL DEFAULT 0,
    max_withdrawal  NUMERIC(36, 18) NOT NULL DEFAULT 0,
    withdrawal_fee  NUMERIC(36, 18) NOT NULL DEFAULT 0,
    deposit_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    withdrawal_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Custody
    custody_mode    settlement_mode NOT NULL DEFAULT 'OFF_CHAIN',
    requires_tag    BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Compliance
    sanctions_screened BOOLEAN NOT NULL DEFAULT FALSE,
    last_screened_at TIMESTAMPTZ,
    
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_assets_symbol ON assets(symbol);
CREATE INDEX idx_assets_type_status ON assets(type, status);
CREATE INDEX idx_assets_network ON assets(network);
CREATE INDEX idx_assets_contract ON assets(contract_address);
CREATE INDEX idx_assets_deleted ON assets(deleted_at) WHERE deleted_at IS NOT NULL;

-- Trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_assets_updated_at BEFORE UPDATE ON assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();