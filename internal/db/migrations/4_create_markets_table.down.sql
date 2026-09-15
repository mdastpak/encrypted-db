-- Migration: 4_create_markets_table.down.sql
-- Description: Drop markets table

DROP TRIGGER IF EXISTS update_markets_updated_at ON markets;
DROP TABLE IF EXISTS market_stats_24h;
DROP TABLE IF EXISTS markets;
DROP TYPE IF EXISTS market_status;