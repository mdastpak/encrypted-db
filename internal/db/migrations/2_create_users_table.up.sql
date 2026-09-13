-- Define the enum type for status
CREATE TYPE USER_STATUS AS ENUM ('disabled', 'deleted', 'unverified', 'suspend', 'approved');

-- Create currencies table
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    hk UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    status USER_STATUS DEFAULT 'unverified',
    info JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

-- Add check constraint to ensure structure of JSONB info
ALTER TABLE users
ADD CONSTRAINT user_info_structure CHECK (
    info ? 'username' AND
    info ? 'password'
);

-- Create a unique index for the `username` field in `info`
CREATE UNIQUE INDEX user_info_username_unique ON currencies ((info->>'username'));

