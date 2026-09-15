-- Migration: 6_create_trades_table.up.sql
-- Description: Create trades table

CREATE TYPE trade_side AS ENUM ('BUY', 'SELL');

CREATE TABLE trades (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    maker_order_id      UUID NOT NULL REFERENCES orders(id),
    taker_order_id      UUID NOT NULL REFERENCES orders(id),
    market_id           UUID NOT NULL REFERENCES markets(id),
    
    side                trade_side NOT NULL,
    price               NUMERIC(36, 18) NOT NULL,
    quantity            NUMERIC(36, 18) NOT NULL,
    
    maker_fee           NUMERIC(36, 18) NOT NULL DEFAULT 0,
    maker_fee_asset_id  UUID NOT NULL REFERENCES assets(id),
    maker_fee_rate_bps  SMALLINT NOT NULL DEFAULT 0,
    
    taker_fee           NUMERIC(36, 18) NOT NULL DEFAULT 0,
    taker_fee_asset_id  UUID NOT NULL REFERENCES assets(id),
    taker_fee_rate_bps  SMALLINT NOT NULL DEFAULT 0,
    
    settlement_mode     settlement_mode NOT NULL DEFAULT 'OFF_CHAIN',
    settled             BOOLEAN NOT NULL DEFAULT FALSE,
    settled_at          TIMESTAMPTZ,
    
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_trades_market_id ON trades(market_id);
CREATE INDEX idx_trades_maker_order ON trades(maker_order_id);
CREATE INDEX idx_trades_taker_order ON trades(taker_order_id);
CREATE INDEX idx_trades_timestamp ON trades(timestamp DESC);
CREATE INDEX idx_trades_market_time ON trades(market_id, timestamp DESC);

-- TimescaleDB hypertable for trades (high volume)
-- SELECT create_hypertable('trades', 'timestamp', chunk_time_interval => INTERVAL '1 day');
-- CREATE INDEX idx_trades_market_time_desc ON trades(market_id, timestamp DESC);

-- Trade events for event sourcing
CREATE TABLE trade_events (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trade_id            UUID NOT NULL REFERENCES trades(id),
    event_type          VARCHAR(50) NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    market_id           UUID NOT NULL REFERENCES markets(id),
    side                trade_side NOT NULL,
    price               NUMERIC(36, 18) NOT NULL,
    quantity            NUMERIC(36, 18) NOT NULL,
    
    maker_order_id      UUID NOT NULL REFERENCES orders(id),
    taker_order_id      UUID NOT NULL REFERENCES orders(id),
    maker_user_id       UUID NOT NULL REFERENCES users(id),
    taker_user_id       UUID NOT NULL REFERENCES users(id),
    
    maker_fee           NUMERIC(36, 18),
    taker_fee           NUMERIC(36, 18),
    
    settlement_mode     settlement_mode
);

CREATE INDEX idx_trade_events_trade_id ON trade_events(trade_id);
CREATE INDEX idx_trade_events_timestamp ON trade_events(timestamp);