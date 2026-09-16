-- Migration: 6_create_trades_table.down.sql
-- Description: Drop trades table

ALTER TABLE order_events DROP CONSTRAINT IF EXISTS fk_order_events_trade_id;
DROP TABLE IF EXISTS trade_events;
DROP TABLE IF EXISTS trades;
DROP TYPE IF EXISTS trade_side;