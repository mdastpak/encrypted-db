-- Define the enum type for status
CREATE TYPE STATUS_ENUM AS ENUM ('approved', 'deleted', 'pending', 'suspend');

-- Create currencies table
CREATE TABLE currencies (
    id SERIAL PRIMARY KEY,
    hk UUID UNIQUE NOT NULL DEFAULT gen_random_uuid(),
    status STATUS_ENUM DEFAULT 'pending',
    info JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    deleted_at TIMESTAMP NULL
);

-- Add check constraint to ensure structure of JSONB info
ALTER TABLE currencies
ADD CONSTRAINT info_structure CHECK (
    info ? 'fa_name' AND
    info ? 'en_name' AND
    info ? 'symbol' AND
    info ? 'if_fiat' AND
    info ? 'is_token' AND
    info ? 'contract'
);

-- Create a unique index for the `symbol` field in `info`
CREATE UNIQUE INDEX currencies_info_symbol_unique ON currencies ((info->>'symbol'));

-- Create a unique index for the `contract` field in `info`
CREATE UNIQUE INDEX currencies_info_contract_unique ON currencies ((info->>'contract'));
