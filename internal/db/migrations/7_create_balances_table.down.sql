-- Migration: 7_create_balances_table.down.sql
-- Description: Drop balances table

DROP TABLE IF EXISTS balance_changes;
DROP TABLE IF EXISTS balance_snapshots;
DROP TABLE IF EXISTS balances;