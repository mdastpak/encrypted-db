-- Migration: 4_create_markets_table.up.sql
-- Description: Create markets table for trading pairs

CREATE TYPE market_status AS ENUM ('ACTIVE', 'INACTIVE', 'MAINTENANCE', 'SUSPENDED', 'DELISTED');

CREATE TABLE markets (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    base_asset_id       UUID NOT NULL REFERENCES assets(id),
    quote_asset_id      UUID NOT NULL REFERENCES assets(id),
    
    status              market_status NOT NULL DEFAULT 'INACTIVE',
    
    -- Trading parameters
    min_order_size      NUMERIC(36, 18) NOT NULL DEFAULT 0,
    max_order_size      NUMERIC(36, 18) NOT NULL DEFAULT 0,
    min_notional        NUMERIC(36, 18) NOT NULL DEFAULT 0,
    tick_size           NUMERIC(36, 18) NOT NULL DEFAULT 0.01,
    lot_size            NUMERIC(36, 18) NOT NULL DEFAULT 0.001,
    
    -- Fee structure (basis points)
    maker_fee_bps       SMALLINT NOT NULL DEFAULT 10,
    taker_fee_bps       SMALLINT NOT NULL DEFAULT 20,
    
    -- Risk limits
    max_position_size   NUMERIC(36, 18) NOT NULL DEFAULT 0,
    max_leverage        SMALLINT NOT NULL DEFAULT 1,
    
    -- Market making
    market_maker_fee_bps SMALLINT NOT NULL DEFAULT 0,
    
    -- Price bands
    price_band_bps      SMALLINT NOT NULL DEFAULT 1000,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    
    CONSTRAINT chk_different_assets CHECK (base_asset_id != quote_asset_id),
    CONSTRAINT uq_market_pair UNIQUE (base_asset_id, quote_asset_id)
);

CREATE INDEX idx_markets_base_asset ON markets(base_asset_id);
CREATE INDEX idx_markets_quote_asset ON markets(quote_asset_id);
CREATE INDEX idx_markets_status ON markets(status);
CREATE INDEX idx_markets_deleted ON markets(deleted_at) WHERE deleted_at IS NOT NULL;

-- 24h stats table (for TimescaleDB hypertable)
CREATE TABLE market_stats_24h (
    market_id           UUID NOT NULL REFERENCES markets(id),
    timestamp           TIMESTAMPTZ NOT NULL,
    
    -- Price
    last_price          NUMERIC(36, 18),
    open_price          NUMERIC(36, 18),
    high_price          NUMERIC(36, 18),
    low_price           NUMERIC(36, 18),
    price_change        NUMERIC(36, 18),
    price_change_pct    NUMERIC(18, 8),
    
    -- Volume
    base_volume         NUMERIC(36, 18),
    quote_volume        NUMERIC(36, 18),
    trade_count         BIGINT,
    
    -- Order book snapshot
    bid_price           NUMERIC(36, 18),
    bid_size            NUMERIC(36, 18),
    ask_price           NUMERIC(36, 18),
    ask_size            NUMERIC(36, 18),
    spread              NUMERIC(36, 18),
    spread_bps          INTEGER,
    
    -- Derivatives
    funding_rate        NUMERIC(18, 8),
    next_funding_time   TIMESTAMPTZ,
    
    PRIMARY KEY (market_id, timestamp)
);

-- TimescaleDB hypertable (will be enabled when TimescaleDB is installed)
-- SELECT create_hypertable('market_stats_24h', 'timestamp', chunk_time_interval => INTERVAL '1 day');
-- CREATE INDEX idx_market_stats_24h_market_time ON market_stats_24h(market_id, timestamp DESC);

CREATE TRIGGER update_markets_updated_at BEFORE UPDATE ON markets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();