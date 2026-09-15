-- Migration: 5_create_orders_table.down.sql
-- Description: Drop orders table

DROP TRIGGER IF EXISTS update_orders_updated_at ON orders;
DROP TABLE IF EXISTS order_events;
DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_side;
DROP TYPE IF EXISTS order_type;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS time_in_force;