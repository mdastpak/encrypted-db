-- Migration: 9_create_kyc_tables.up.sql
-- Description: Create KYC, AML, and sanctions tables

CREATE TYPE document_type AS ENUM ('PASSPORT', 'NATIONAL_ID', 'DRIVERS_LICENSE', 'UTILITY_BILL', 'BANK_STATEMENT', 'PROOF_OF_ADDRESS', 'SELFIE', 'SOURCE_OF_FUNDS', 'CORPORATE_DOCUMENT');
CREATE TYPE document_status AS ENUM ('PENDING', 'APPROVED', 'REJECTED', 'EXPIRED');
CREATE TYPE screen_type AS ENUM ('USER_ONBOARDING', 'DEPOSIT_ADDRESS', 'WITHDRAWAL_ADDRESS', 'COUNTERPARTY', 'PERIODIC_REVIEW', 'TRANSACTION');
CREATE TYPE aml_rule_type AS ENUM ('VELOCITY', 'STRUCTURING', 'HIGH_RISK_COUNTRY', 'MIXER', 'DARKNET', 'SANCTIONS', 'PEP', 'UNUSUAL_PATTERN');
CREATE TYPE aml_action AS ENUM ('ALERT', 'REVIEW', 'BLOCK', 'FREEZE');
CREATE TYPE aml_alert_status AS ENUM ('OPEN', 'INVESTIGATING', 'RESOLVED', 'DISMISSED', 'ESCALATED');
CREATE TYPE sanction_action AS ENUM ('ALLOW', 'REVIEW', 'BLOCK');

-- KYC Profiles
CREATE TABLE kyc_profiles (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                         UUID NOT NULL UNIQUE REFERENCES users(hk) ON DELETE CASCADE,
    
    status                          kyc_status NOT NULL DEFAULT 'PENDING',
    tier                            kyc_tier NOT NULL DEFAULT 'NONE',
    
    -- Personal info (encrypted)
    first_name_enc                  TEXT,
    last_name_enc                   TEXT,
    date_of_birth_enc               DATE,
    nationality_enc                 VARCHAR(100),
    country_of_residence_enc        VARCHAR(100),
    
    address_line1_enc               TEXT,
    address_line2_enc               TEXT,
    city_enc                        VARCHAR(100),
    state_enc                       VARCHAR(100),
    postal_code_enc                 VARCHAR(20),
    country_enc                     VARCHAR(100),
    
    verified_at                     TIMESTAMPTZ,
    verified_by                     UUID REFERENCES users(hk),
    expires_at                      TIMESTAMPTZ,
    
    risk_score                      SMALLINT NOT NULL DEFAULT 0,
    risk_factors                    TEXT[] DEFAULT '{}',
    
    provider                        VARCHAR(50),
    provider_ref_id                 VARCHAR(100),
    
    admin_notes                     TEXT,
    reject_reason                   TEXT,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kyc_profiles_user_id ON kyc_profiles(user_id);
CREATE INDEX idx_kyc_profiles_status ON kyc_profiles(status);
CREATE INDEX idx_kyc_profiles_tier ON kyc_profiles(tier);

CREATE TRIGGER update_kyc_profiles_updated_at BEFORE UPDATE ON kyc_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- KYC Documents
CREATE TABLE kyc_documents (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    kyc_profile_id                  UUID NOT NULL REFERENCES kyc_profiles(id) ON DELETE CASCADE,
    
    type                            document_type NOT NULL,
    status                          document_status NOT NULL DEFAULT 'PENDING',
    
    file_name                       VARCHAR(255) NOT NULL,
    file_size                       BIGINT NOT NULL,
    mime_type                       VARCHAR(100) NOT NULL,
    storage_path                    VARCHAR(500) NOT NULL,
    
    verified_at                     TIMESTAMPTZ,
    verified_by                     UUID REFERENCES users(hk),
    reject_reason                   TEXT,
    
    extracted_data_enc              TEXT,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kyc_documents_profile_id ON kyc_documents(kyc_profile_id);
CREATE INDEX idx_kyc_documents_status ON kyc_documents(status);

CREATE TRIGGER update_kyc_documents_updated_at BEFORE UPDATE ON kyc_documents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- KYC Applications
CREATE TABLE kyc_applications (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                         UUID NOT NULL REFERENCES users(hk) ON DELETE CASCADE,
    requested_tier                  kyc_tier NOT NULL,
    
    status                          kyc_status NOT NULL DEFAULT 'PENDING',
    
    form_data_enc                   TEXT,
    
    documents                       UUID[] DEFAULT '{}',
    
    reviewed_at                     TIMESTAMPTZ,
    reviewed_by                     UUID REFERENCES users(hk),
    review_notes                    TEXT,
    
    provider                        VARCHAR(50),
    provider_ref_id                 VARCHAR(100),
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_kyc_applications_user_id ON kyc_applications(user_id);
CREATE INDEX idx_kyc_applications_status ON kyc_applications(status);

CREATE TRIGGER update_kyc_applications_updated_at BEFORE UPDATE ON kyc_applications
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sanctions Screenings
CREATE TABLE sanctions_screenings (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id                         UUID REFERENCES users(hk) ON DELETE SET NULL,
    address_id                      UUID,
    counterparty_id                 UUID,
    
    screen_type                     screen_type NOT NULL,
    provider                        VARCHAR(50) NOT NULL,
    provider_ref_id                 VARCHAR(100),
    
    action                          sanction_action NOT NULL,
    risk_score                      NUMERIC(5, 2) NOT NULL DEFAULT 0,
    matched_lists                   TEXT[] DEFAULT '{}',
    matched_entries                 TEXT[] DEFAULT '{}',
    
    decided_at                      TIMESTAMPTZ,
    decided_by                      UUID REFERENCES users(hk),
    notes                           TEXT,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sanctions_user_id ON sanctions_screenings(user_id);
CREATE INDEX idx_sanctions_action ON sanctions_screenings(action);
CREATE INDEX idx_sanctions_screen_type ON sanctions_screenings(screen_type);
CREATE INDEX idx_sanctions_created ON sanctions_screenings(created_at);

-- AML Rules
CREATE TABLE aml_rules (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name                            VARCHAR(100) NOT NULL,
    description                     TEXT,
    
    rule_type                       aml_rule_type NOT NULL,
    config                          JSONB NOT NULL DEFAULT '{}',
    
    threshold_amount                NUMERIC(36, 18),
    threshold_count                 INTEGER,
    time_window_hours               INTEGER,
    
    action                          aml_action NOT NULL DEFAULT 'ALERT',
    severity                        VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    
    enabled                         BOOLEAN NOT NULL DEFAULT TRUE,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_rules_type ON aml_rules(rule_type);
CREATE INDEX idx_aml_rules_enabled ON aml_rules(enabled);

CREATE TRIGGER update_aml_rules_updated_at BEFORE UPDATE ON aml_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- AML Alerts
CREATE TABLE aml_alerts (
    id                              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id                         UUID NOT NULL REFERENCES aml_rules(id),
    user_id                         UUID NOT NULL REFERENCES users(hk),
    sub_account_id                  UUID REFERENCES sub_accounts(id),
    
    trigger_data                    JSONB NOT NULL,
    risk_score                      NUMERIC(5, 2) NOT NULL DEFAULT 0,
    
    status                          aml_alert_status NOT NULL DEFAULT 'OPEN',
    assigned_to                     UUID REFERENCES users(hk),
    resolved_at                     TIMESTAMPTZ,
    resolved_by                     UUID REFERENCES users(hk),
    resolution                      TEXT,
    
    created_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_aml_alerts_rule_id ON aml_alerts(rule_id);
CREATE INDEX idx_aml_alerts_user_id ON aml_alerts(user_id);
CREATE INDEX idx_aml_alerts_status ON aml_alerts(status);
CREATE INDEX idx_aml_alerts_created ON aml_alerts(created_at);

CREATE TRIGGER update_aml_alerts_updated_at BEFORE UPDATE ON aml_alerts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Sanction action enum (created above)