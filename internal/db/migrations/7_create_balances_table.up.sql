-- Migration: 7_create_balances_table.up.sql
-- Description: Create balances table

CREATE TABLE balances (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id             UUID NOT NULL REFERENCES users(id),
    sub_account_id      UUID REFERENCES sub_accounts(id),
    asset_id            UUID NOT NULL REFERENCES assets(id),
    
    available           NUMERIC(36, 18) NOT NULL DEFAULT 0,
    locked              NUMERIC(36, 18) NOT NULL DEFAULT 0,
    on_chain            NUMERIC(36, 18) NOT NULL DEFAULT 0,
    pending_deposit     NUMERIC(36, 18) NOT NULL DEFAULT 0,
    
    total               NUMERIC(36, 18) NOT NULL DEFAULT 0,
    
    version             INTEGER NOT NULL DEFAULT 1,
    
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_user_asset_sub UNIQUE (user_id, sub_account_id, asset_id)
);

CREATE INDEX idx_balances_user_id ON balances(user_id);
CREATE INDEX idx_balances_sub_account_id ON balances(sub_account_id);
CREATE INDEX idx_balances_asset_id ON balances(asset_id);
CREATE INDEX idx_balances_total ON balances(total DESC);

CREATE TRIGGER update_balances_updated_at BEFORE UPDATE ON balances
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Balance snapshots for audit
CREATE TABLE balance_snapshots (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    balance_id          UUID NOT NULL REFERENCES balances(id),
    user_id             UUID NOT NULL REFERENCES users(id),
    sub_account_id      UUID REFERENCES sub_accounts(id),
    asset_id            UUID NOT NULL REFERENCES assets(id),
    
    available           NUMERIC(36, 18) NOT NULL,
    locked              NUMERIC(36, 18) NOT NULL,
    on_chain            NUMERIC(36, 18) NOT NULL,
    pending_deposit     NUMERIC(36, 18) NOT NULL,
    total               NUMERIC(36, 18) NOT NULL,
    version             INTEGER NOT NULL,
    
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_balance_snapshots_balance_id ON balance_snapshots(balance_id);
CREATE INDEX idx_balance_snapshots_timestamp ON balance_snapshots(timestamp);

-- Balance change events for event sourcing
CREATE TABLE balance_changes (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    balance_id          UUID NOT NULL REFERENCES balances(id),
    user_id             UUID NOT NULL REFERENCES users(id),
    sub_account_id      UUID REFERENCES sub_accounts(id),
    asset_id            UUID NOT NULL REFERENCES assets(id),
    
    change_type         VARCHAR(50) NOT NULL,
    amount              NUMERIC(36, 18) NOT NULL,
    
    old_available       NUMERIC(36, 18) NOT NULL,
    new_available       NUMERIC(36, 18) NOT NULL,
    old_locked          NUMERIC(36, 18) NOT NULL,
    new_locked          NUMERIC(36, 18) NOT NULL,
    old_on_chain        NUMERIC(36, 18) NOT NULL,
    new_on_chain        NUMERIC(36, 18) NOT NULL,
    old_pending         NUMERIC(36, 18) NOT NULL,
    new_pending         NUMERIC(36, 18) NOT NULL,
    
    reference_id        UUID,
    reference_type      VARCHAR(50),
    
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_balance_changes_balance_id ON balance_changes(balance_id);
CREATE INDEX idx_balance_changes_user_id ON balance_changes(user_id);
CREATE INDEX idx_balance_changes_timestamp ON balance_changes(timestamp);
CREATE INDEX idx_balance_changes_reference ON balance_changes(reference_id, reference_type);