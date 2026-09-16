-- Migration: 5_create_orders_table.up.sql
-- Description: Create orders table

CREATE TYPE order_side AS ENUM ('BUY', 'SELL');
CREATE TYPE order_type AS ENUM ('MARKET', 'LIMIT', 'STOP', 'STOP_LIMIT');
CREATE TYPE order_status AS ENUM ('NEW', 'PARTIALLY_FILLED', 'FILLED', 'CANCELLED', 'REJECTED', 'EXPIRED');
CREATE TYPE time_in_force AS ENUM ('GTC', 'IOC', 'FOK', 'GTX');

CREATE TABLE orders (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    client_order_id     VARCHAR(64),
    
    user_id             UUID NOT NULL REFERENCES users(hk),
    sub_account_id      UUID,
    market_id           UUID NOT NULL REFERENCES markets(id),
    
    side                order_side NOT NULL,
    type                order_type NOT NULL,
    time_in_force       time_in_force NOT NULL DEFAULT 'GTC',
    
    price               NUMERIC(36, 18) NOT NULL DEFAULT 0,
    stop_price          NUMERIC(36, 18),
    quantity            NUMERIC(36, 18) NOT NULL,
    filled_quantity     NUMERIC(36, 18) NOT NULL DEFAULT 0,
    
    status              order_status NOT NULL DEFAULT 'NEW',
    reject_reason       TEXT,
    
    fee_asset_id        UUID NOT NULL REFERENCES assets(id),
    fee_paid            NUMERIC(36, 18) NOT NULL DEFAULT 0,
    fee_rate_bps        SMALLINT NOT NULL DEFAULT 0,
    
    reduce_only         BOOLEAN NOT NULL DEFAULT FALSE,
    post_only           BOOLEAN NOT NULL DEFAULT FALSE,
    
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expired_at          TIMESTAMPTZ,
    cancelled_at        TIMESTAMPTZ,
    
    CONSTRAINT uq_client_order_id UNIQUE (user_id, client_order_id)
);

CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_sub_account_id ON orders(sub_account_id);
CREATE INDEX idx_orders_market_id ON orders(market_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_created_at ON orders(created_at DESC);
CREATE INDEX idx_orders_market_status ON orders(market_id, status);
CREATE INDEX idx_orders_client_order_id ON orders(client_order_id);

CREATE TRIGGER update_orders_updated_at BEFORE UPDATE ON orders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Order events table for event sourcing
CREATE TABLE order_events (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    order_id            UUID NOT NULL REFERENCES orders(id),
    event_type          VARCHAR(50) NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    old_status          order_status,
    new_status          order_status,
    old_filled          NUMERIC(36, 18),
    new_filled          NUMERIC(36, 18),
    
    trade_id            UUID,
    trade_price         NUMERIC(36, 18),
    trade_qty           NUMERIC(36, 18),
    trade_fee           NUMERIC(36, 18),
    
    metadata            JSONB
);

CREATE INDEX idx_order_events_order_id ON order_events(order_id);
CREATE INDEX idx_order_events_timestamp ON order_events(timestamp);