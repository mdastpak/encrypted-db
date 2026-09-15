-- Migration: 3_create_assets_table.down.sql
-- Description: Drop assets table

DROP TRIGGER IF EXISTS update_assets_updated_at ON assets;
DROP TABLE IF EXISTS assets;
DROP TYPE IF EXISTS asset_type;
DROP TYPE IF EXISTS asset_status;
DROP TYPE IF EXISTS settlement_mode;