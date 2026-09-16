-- Drop the users table and enum type
ALTER TABLE users DROP CONSTRAINT IF EXISTS user_info_structure;
DROP INDEX IF EXISTS user_info_username_unique;
DROP TABLE IF EXISTS users;
DROP TYPE IF EXISTS USER_STATUS;
