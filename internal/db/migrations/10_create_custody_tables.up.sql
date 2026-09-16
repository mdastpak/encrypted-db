-- Migration: 10_create_custody_tables.up.sql
-- Description: Create custody tables for deposit addresses, withdrawals, settlements

CREATE TYPE custody_provider_type AS ENUM ('INTERNAL', 'BTC', 'EVM', 'SOLANA', 'FIAT');
CREATE TYPE deposit_status_type AS ENUM ('DETECTED', 'CONFIRMING', 'CONFIRMED', 'CREDITED', 'FAILED');
CREATE TYPE withdrawal_status_type AS ENUM ('PENDING', 'BROADCAST', 'CONFIRMING', 'COMPLETED', 'FAILED', 'CANCELLED', 'REJECTED');
CREATE TYPE fee_level AS ENUM ('SLOW', 'NORMAL', 'FAST', 'CUSTOM');
CREATE TYPE settlement_type AS ENUM ('TRADE', 'WITHDRAWAL', 'DEPOSIT', 'TRANSFER', 'FEE', 'FUNDING', 'LIQUIDATION');
CREATE TYPE settlement_status AS ENUM ('PENDING', 'PROCESSING', 'COMPLETED', 'FAILED', 'CANCELLED');

-- Deposit Addresses
CREATE TABLE deposit_addresses (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                         UUID NOT NULL REFERENCES users(hk) ON DELETE CASCADE,
    sub_account_id                  UUID REFERENCES sub_accounts(id) ON DELETE CASCADE,
    asset_id                        UUID NOT NULL REFERENCES assets(id),
    
    address                         VARCHAR(200) NOT NULL,
    tag                             VARCHAR(100),
    
    -- HD wallet info
    derivation_path                 VARCHAR(100),
    index                           INTEGER,
    
    active                          BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_user_asset_address UNIQUE (user_id, sub_account_id, asset_id, address)
);

CREATE INDEX idx_deposit_addresses_user_id ON deposit_addresses(user_id);
CREATE INDEX idx_deposit_addresses_asset_id ON deposit_addresses(asset_id);
CREATE INDEX idx_deposit_addresses_address ON deposit_addresses(address);

-- Deposits (detected on-chain deposits)
CREATE TABLE deposits (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tx_hash                         VARCHAR(200) NOT NULL,
    asset_id                        UUID NOT NULL REFERENCES assets(id),
    address                         VARCHAR(200) NOT NULL,
    tag                             VARCHAR(100),
    
    amount                          NUMERIC(36, 18) NOT NULL,
    confirmed_amount                NUMERIC(36, 18) NOT NULL DEFAULT 0,
    
    confirmations                   INTEGER NOT NULL DEFAULT 0,
    required_confirms               INTEGER NOT NULL DEFAULT 1,
    
    status                          deposit_status_type NOT NULL DEFAULT 'DETECTED',
    
    detected_at                     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at                    TIMESTAMPTZ,
    credited_at                     TIMESTAMPTZ,
    
    block_height                    BIGINT,
    block_hash                      VARCHAR(100),
    raw_data                        JSONB,
    
    CONSTRAINT uq_deposit_tx_asset UNIQUE (tx_hash, asset_id)
);

CREATE INDEX idx_deposits_tx_hash ON deposits(tx_hash);
CREATE INDEX idx_deposits_address ON deposits(address);
CREATE INDEX idx_deposits_status ON deposits(status);
CREATE INDEX idx_deposits_asset_id ON deposits(asset_id);
CREATE INDEX idx_deposits_detected_at ON deposits(detected_at);

-- Withdrawals
CREATE TABLE withdrawals (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                         UUID NOT NULL REFERENCES users(hk) ON DELETE CASCADE,
    sub_account_id                  UUID REFERENCES sub_accounts(id) ON DELETE CASCADE,
    asset_id                        UUID NOT NULL REFERENCES assets(id),
    
    amount                          NUMERIC(36, 18) NOT NULL,
    address                         VARCHAR(200) NOT NULL,
    tag                             VARCHAR(100),
    
    fee_level                       fee_level NOT NULL DEFAULT 'NORMAL',
    custom_fee                      NUMERIC(36, 18),
    subtract_fee                    BOOLEAN NOT NULL DEFAULT FALSE,
    priority                        BOOLEAN NOT NULL DEFAULT FALSE,
    
    idempotency_key                 VARCHAR(100) NOT NULL UNIQUE,
    
    status                          withdrawal_status_type NOT NULL DEFAULT 'PENDING',
    tx_hash                         VARCHAR(200),
    
    estimated_fee                   NUMERIC(36, 18) NOT NULL DEFAULT 0,
    actual_fee                      NUMERIC(36, 18) NOT NULL DEFAULT 0,
    net_amount                      NUMERIC(36, 18) NOT NULL DEFAULT 0,
    
    confirmations                   INTEGER NOT NULL DEFAULT 0,
    required_confirms               INTEGER NOT NULL DEFAULT 1,
    
    broadcast_at                    TIMESTAMPTZ,
    confirmed_at                    TIMESTAMPTZ,
    
    error_message                   TEXT,
    raw_data                        JSONB,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_withdrawals_user_id ON withdrawals(user_id);
CREATE INDEX idx_withdrawals_asset_id ON withdrawals(asset_id);
CREATE INDEX idx_withdrawals_status ON withdrawals(status);
CREATE INDEX idx_withdrawals_idempotency ON withdrawals(idempotency_key);
CREATE INDEX idx_withdrawals_created ON withdrawals(created_at);

CREATE TRIGGER update_withdrawals_updated_at BEFORE UPDATE ON withdrawals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Settlements (internal and on-chain)
CREATE TABLE settlements (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type                            settlement_type NOT NULL,
    
    from_user_id                    UUID NOT NULL REFERENCES users(hk),
    to_user_id                      UUID NOT NULL REFERENCES users(hk),
    from_sub_account_id             UUID REFERENCES sub_accounts(id),
    to_sub_account_id               UUID REFERENCES sub_accounts(id),
    
    asset_id                        UUID NOT NULL REFERENCES assets(id),
    amount                          NUMERIC(36, 18) NOT NULL,
    
    mode                            settlement_mode NOT NULL DEFAULT 'OFF_CHAIN',
    
    from_address                    VARCHAR(200),
    to_address                      VARCHAR(200),
    tx_hash                         VARCHAR(200),
    
    reference_id                    UUID,
    reference_type                  VARCHAR(50),
    
    status                          settlement_status NOT NULL DEFAULT 'PENDING',
    error_message                   TEXT,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    processed_at                    TIMESTAMPTZ,
    completed_at                    TIMESTAMPTZ
);

CREATE INDEX idx_settlements_from_user ON settlements(from_user_id);
CREATE INDEX idx_settlements_to_user ON settlements(to_user_id);
CREATE INDEX idx_settlements_asset ON settlements(asset_id);
CREATE INDEX idx_settlements_status ON settlements(status);
CREATE INDEX idx_settlements_reference ON settlements(reference_id, reference_type);
CREATE INDEX idx_settlements_type ON settlements(type);
CREATE INDEX idx_settlements_created ON settlements(created_at);

-- Custody provider configurations
CREATE TABLE custody_configs (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider_type                   custody_provider_type NOT NULL,
    asset_id                        UUID NOT NULL REFERENCES assets(id),
    network                         VARCHAR(50) NOT NULL,
    
    -- Connection
    rpc_endpoints                   TEXT[] NOT NULL DEFAULT '{}',
    ws_endpoints                    TEXT[] DEFAULT '{}',
    explorer_api                    VARCHAR(500),
    explorer_ws                     VARCHAR(500),
    
    -- Auth
    api_key                         VARCHAR(200),
    api_secret                      VARCHAR(500),
    jwt_token                       TEXT,
    
    -- Chain
    chain_id                        VARCHAR(50),
    contract_address                VARCHAR(100),
    decimals                        SMALLINT NOT NULL DEFAULT 18,
    
    -- Deposit
    confirmations_required          INTEGER NOT NULL DEFAULT 1,
    min_deposit_amount              NUMERIC(36, 18) NOT NULL DEFAULT 0,
    
    -- Withdrawal
    min_withdrawal_amount           NUMERIC(36, 18) NOT NULL DEFAULT 0,
    max_withdrawal_amount           NUMERIC(36, 18) NOT NULL DEFAULT 0,
    default_fee_level               fee_level NOT NULL DEFAULT 'NORMAL',
    fee_asset_id                    UUID REFERENCES assets(id),
    
    -- Wallets
    hot_wallet_address              VARCHAR(200),
    cold_wallet_address             VARCHAR(200),
    
    -- Monitoring
    block_poll_interval             INTEGER NOT NULL DEFAULT 30,
    reorg_depth                     INTEGER NOT NULL DEFAULT 1,
    
    -- Compliance
    sanctions_screening             BOOLEAN NOT NULL DEFAULT TRUE,
    
    metadata                        JSONB DEFAULT '{}',
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_custody_asset_network UNIQUE (asset_id, network)
);

CREATE INDEX idx_custody_configs_asset ON custody_configs(asset_id);
CREATE INDEX idx_custody_configs_provider ON custody_configs(provider_type);

CREATE TRIGGER update_custody_configs_updated_at BEFORE UPDATE ON custody_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();