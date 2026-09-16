-- Migration: 6_create_trades_table.down.sql
-- Description: Drop trades table

DROP TABLE IF EXISTS trade_events;
DROP TABLE IF EXISTS trades;
DROP TYPE IF EXISTS trade_side;