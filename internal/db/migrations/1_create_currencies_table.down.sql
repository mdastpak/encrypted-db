-- Drop the currencies table and enum type
ALTER TABLE currencies DROP CONSTRAINT IF EXISTS info_structure;
DROP TABLE IF EXISTS currencies;
DROP TYPE IF EXISTS STATUS_ENUM;
