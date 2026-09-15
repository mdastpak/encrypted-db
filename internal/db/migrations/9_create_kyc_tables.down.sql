-- Migration: 9_create_kyc_tables.down.sql
-- Description: Drop KYC tables

DROP TRIGGER IF EXISTS update_kyc_profiles_updated_at ON kyc_profiles;
DROP TRIGGER IF EXISTS update_kyc_documents_updated_at ON kyc_documents;
DROP TRIGGER IF EXISTS update_kyc_applications_updated_at ON kyc_applications;
DROP TRIGGER IF EXISTS update_aml_rules_updated_at ON aml_rules;
DROP TRIGGER IF EXISTS update_aml_alerts_updated_at ON aml_alerts;

DROP TABLE IF EXISTS aml_alerts;
DROP TABLE IF EXISTS aml_rules;
DROP TABLE IF EXISTS sanctions_screenings;
DROP TABLE IF EXISTS kyc_applications;
DROP TABLE IF EXISTS kyc_documents;
DROP TABLE IF EXISTS kyc_profiles;

DROP TYPE IF EXISTS document_type;
DROP TYPE IF EXISTS document_status;
DROP TYPE IF EXISTS screen_type;
DROP TYPE IF EXISTS aml_rule_type;
DROP TYPE IF EXISTS aml_action;
DROP TYPE IF EXISTS aml_alert_status;