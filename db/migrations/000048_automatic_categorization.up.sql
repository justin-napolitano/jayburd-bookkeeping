-- #488: attributed analytic categories and durable automatic decisions; never journals.
BEGIN;
SET LOCAL ROLE bookkeeping_owner;
LOCK TABLE banking.collection_workspace,banking.collection_label_state IN SHARE ROW EXCLUSIVE MODE;

-- Minimal interoperability codes and application labels; no vendor descriptions or merchant dictionary.
CREATE TABLE reference.analytic_category(
 code text PRIMARY KEY CHECK(code COLLATE "C" ~ '^[A-Z][A-Z_]{0,127}$'),
 topic_id uuid NOT NULL UNIQUE,name text NOT NULL CHECK(length(name) BETWEEN 1 AND 80),
 map_edition text NOT NULL,source_sha256 text NOT NULL CHECK(source_sha256 COLLATE "C" ~ '^[0-9a-f]{64}$')
);
CREATE TABLE reference.analytic_category_pair(
 version text NOT NULL CHECK(version IN('v1','v2')),primary_code text NOT NULL REFERENCES reference.analytic_category,
 detailed_code text NOT NULL CHECK(detailed_code COLLATE "C" ~ '^[A-Z][A-Z0-9_]{0,127}$'),
 PRIMARY KEY(version,primary_code,detailed_code)
);
INSERT INTO reference.analytic_category VALUES
('BANK_FEES','a4880000-0000-4000-8000-000000000001','Bank fees','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('ENTERTAINMENT','a4880000-0000-4000-8000-000000000002','Entertainment','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('FOOD_AND_DRINK','a4880000-0000-4000-8000-000000000003','Food and drink','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('GENERAL_MERCHANDISE','a4880000-0000-4000-8000-000000000004','General merchandise','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('GENERAL_SERVICES','a4880000-0000-4000-8000-000000000005','General services','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('GOVERNMENT_AND_NON_PROFIT','a4880000-0000-4000-8000-000000000006','Government and nonprofit','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('HOME_IMPROVEMENT','a4880000-0000-4000-8000-000000000007','Home improvement','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('INCOME','a4880000-0000-4000-8000-000000000008','Income','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('LOAN_DISBURSEMENTS','a4880000-0000-4000-8000-000000000009','Loan disbursements','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('LOAN_PAYMENTS','a4880000-0000-4000-8000-000000000010','Loan payments','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('MEDICAL','a4880000-0000-4000-8000-000000000011','Medical','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('OTHER','a4880000-0000-4000-8000-000000000012','Other','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('PERSONAL_CARE','a4880000-0000-4000-8000-000000000013','Personal care','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('RENT_AND_UTILITIES','a4880000-0000-4000-8000-000000000014','Rent and utilities','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('TRANSFER_IN','a4880000-0000-4000-8000-000000000015','Transfers in','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('TRANSFER_OUT','a4880000-0000-4000-8000-000000000016','Transfers out','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('TRANSPORTATION','a4880000-0000-4000-8000-000000000017','Transportation','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc'),
('TRAVEL','a4880000-0000-4000-8000-000000000018','Travel','plaid-pfc-primary-2026-11-11','c585fae8e48b834f78791c13c1fd013ae8594df337cf27f4de85bf9cae89f1bc');
INSERT INTO reference.analytic_category_pair VALUES
('v1','BANK_FEES','BANK_FEES_ATM_FEES'),
('v1','BANK_FEES','BANK_FEES_FOREIGN_TRANSACTION_FEES'),
('v1','BANK_FEES','BANK_FEES_INSUFFICIENT_FUNDS'),
('v1','BANK_FEES','BANK_FEES_INTEREST_CHARGE'),
('v1','BANK_FEES','BANK_FEES_OTHER_BANK_FEES'),
('v1','BANK_FEES','BANK_FEES_OVERDRAFT_FEES'),
('v1','ENTERTAINMENT','ENTERTAINMENT_CASINOS_AND_GAMBLING'),
('v1','ENTERTAINMENT','ENTERTAINMENT_MUSIC_AND_AUDIO'),
('v1','ENTERTAINMENT','ENTERTAINMENT_OTHER_ENTERTAINMENT'),
('v1','ENTERTAINMENT','ENTERTAINMENT_SPORTING_EVENTS_AMUSEMENT_PARKS_AND_MUSEUMS'),
('v1','ENTERTAINMENT','ENTERTAINMENT_TV_AND_MOVIES'),
('v1','ENTERTAINMENT','ENTERTAINMENT_VIDEO_GAMES'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_BEER_WINE_AND_LIQUOR'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_COFFEE'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_FAST_FOOD'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_GROCERIES'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_OTHER_FOOD_AND_DRINK'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_RESTAURANT'),
('v1','FOOD_AND_DRINK','FOOD_AND_DRINK_VENDING_MACHINES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_BOOKSTORES_AND_NEWSSTANDS'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_CLOTHING_AND_ACCESSORIES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_CONVENIENCE_STORES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_DEPARTMENT_STORES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_DISCOUNT_STORES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_ELECTRONICS'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_GIFTS_AND_NOVELTIES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_OFFICE_SUPPLIES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_ONLINE_MARKETPLACES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_OTHER_GENERAL_MERCHANDISE'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_PET_SUPPLIES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_SPORTING_GOODS'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_SUPERSTORES'),
('v1','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_TOBACCO_AND_VAPE'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_ACCOUNTING_AND_FINANCIAL_PLANNING'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_AUTOMOTIVE'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_CHILDCARE'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_CONSULTING_AND_LEGAL'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_EDUCATION'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_INSURANCE'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_OTHER_GENERAL_SERVICES'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_POSTAGE_AND_SHIPPING'),
('v1','GENERAL_SERVICES','GENERAL_SERVICES_STORAGE'),
('v1','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_DONATIONS'),
('v1','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_GOVERNMENT_DEPARTMENTS_AND_AGENCIES'),
('v1','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_OTHER_GOVERNMENT_AND_NON_PROFIT'),
('v1','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_TAX_PAYMENT'),
('v1','HOME_IMPROVEMENT','HOME_IMPROVEMENT_FURNITURE'),
('v1','HOME_IMPROVEMENT','HOME_IMPROVEMENT_HARDWARE'),
('v1','HOME_IMPROVEMENT','HOME_IMPROVEMENT_OTHER_HOME_IMPROVEMENT'),
('v1','HOME_IMPROVEMENT','HOME_IMPROVEMENT_REPAIR_AND_MAINTENANCE'),
('v1','HOME_IMPROVEMENT','HOME_IMPROVEMENT_SECURITY'),
('v1','INCOME','INCOME_DIVIDENDS'),
('v1','INCOME','INCOME_INTEREST_EARNED'),
('v1','INCOME','INCOME_OTHER_INCOME'),
('v1','INCOME','INCOME_RETIREMENT_PENSION'),
('v1','INCOME','INCOME_TAX_REFUND'),
('v1','INCOME','INCOME_UNEMPLOYMENT'),
('v1','INCOME','INCOME_WAGES'),
('v1','LOAN_PAYMENTS','LOAN_PAYMENTS_CAR_PAYMENT'),
('v1','LOAN_PAYMENTS','LOAN_PAYMENTS_CREDIT_CARD_PAYMENT'),
('v1','LOAN_PAYMENTS','LOAN_PAYMENTS_MORTGAGE_PAYMENT'),
('v1','LOAN_PAYMENTS','LOAN_PAYMENTS_OTHER_PAYMENT'),
('v1','LOAN_PAYMENTS','LOAN_PAYMENTS_PERSONAL_LOAN_PAYMENT'),
('v1','LOAN_PAYMENTS','LOAN_PAYMENTS_STUDENT_LOAN_PAYMENT'),
('v1','MEDICAL','MEDICAL_DENTAL_CARE'),
('v1','MEDICAL','MEDICAL_EYE_CARE'),
('v1','MEDICAL','MEDICAL_NURSING_CARE'),
('v1','MEDICAL','MEDICAL_OTHER_MEDICAL'),
('v1','MEDICAL','MEDICAL_PHARMACIES_AND_SUPPLEMENTS'),
('v1','MEDICAL','MEDICAL_PRIMARY_CARE'),
('v1','MEDICAL','MEDICAL_VETERINARY_SERVICES'),
('v1','PERSONAL_CARE','PERSONAL_CARE_GYMS_AND_FITNESS_CENTERS'),
('v1','PERSONAL_CARE','PERSONAL_CARE_HAIR_AND_BEAUTY'),
('v1','PERSONAL_CARE','PERSONAL_CARE_LAUNDRY_AND_DRY_CLEANING'),
('v1','PERSONAL_CARE','PERSONAL_CARE_OTHER_PERSONAL_CARE'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_GAS_AND_ELECTRICITY'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_INTERNET_AND_CABLE'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_OTHER_UTILITIES'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_RENT'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_SEWAGE_AND_WASTE_MANAGEMENT'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_TELEPHONE'),
('v1','RENT_AND_UTILITIES','RENT_AND_UTILITIES_WATER'),
('v1','TRANSFER_IN','TRANSFER_IN_ACCOUNT_TRANSFER'),
('v1','TRANSFER_IN','TRANSFER_IN_CASH_ADVANCES_AND_LOANS'),
('v1','TRANSFER_IN','TRANSFER_IN_DEPOSIT'),
('v1','TRANSFER_IN','TRANSFER_IN_INVESTMENT_AND_RETIREMENT_FUNDS'),
('v1','TRANSFER_IN','TRANSFER_IN_OTHER_TRANSFER_IN'),
('v1','TRANSFER_IN','TRANSFER_IN_SAVINGS'),
('v1','TRANSFER_OUT','TRANSFER_OUT_ACCOUNT_TRANSFER'),
('v1','TRANSFER_OUT','TRANSFER_OUT_INVESTMENT_AND_RETIREMENT_FUNDS'),
('v1','TRANSFER_OUT','TRANSFER_OUT_OTHER_TRANSFER_OUT'),
('v1','TRANSFER_OUT','TRANSFER_OUT_SAVINGS'),
('v1','TRANSFER_OUT','TRANSFER_OUT_WITHDRAWAL'),
('v1','TRANSPORTATION','TRANSPORTATION_BIKES_AND_SCOOTERS'),
('v1','TRANSPORTATION','TRANSPORTATION_GAS'),
('v1','TRANSPORTATION','TRANSPORTATION_OTHER_TRANSPORTATION'),
('v1','TRANSPORTATION','TRANSPORTATION_PARKING'),
('v1','TRANSPORTATION','TRANSPORTATION_PUBLIC_TRANSIT'),
('v1','TRANSPORTATION','TRANSPORTATION_TAXIS_AND_RIDE_SHARES'),
('v1','TRANSPORTATION','TRANSPORTATION_TOLLS'),
('v1','TRAVEL','TRAVEL_FLIGHTS'),
('v1','TRAVEL','TRAVEL_LODGING'),
('v1','TRAVEL','TRAVEL_OTHER_TRAVEL'),
('v1','TRAVEL','TRAVEL_RENTAL_CARS'),
('v2','BANK_FEES','BANK_FEES_ATM_FEES'),
('v2','BANK_FEES','BANK_FEES_CASH_ADVANCE'),
('v2','BANK_FEES','BANK_FEES_FOREIGN_TRANSACTION_FEES'),
('v2','BANK_FEES','BANK_FEES_INSUFFICIENT_FUNDS'),
('v2','BANK_FEES','BANK_FEES_INTEREST_CHARGE'),
('v2','BANK_FEES','BANK_FEES_LATE_FEES'),
('v2','BANK_FEES','BANK_FEES_OTHER_BANK_FEES'),
('v2','BANK_FEES','BANK_FEES_OVERDRAFT_FEES'),
('v2','ENTERTAINMENT','ENTERTAINMENT_CASINOS_AND_GAMBLING'),
('v2','ENTERTAINMENT','ENTERTAINMENT_MUSIC_AND_AUDIO'),
('v2','ENTERTAINMENT','ENTERTAINMENT_OTHER_ENTERTAINMENT'),
('v2','ENTERTAINMENT','ENTERTAINMENT_SPORTING_EVENTS_AMUSEMENT_PARKS_AND_MUSEUMS'),
('v2','ENTERTAINMENT','ENTERTAINMENT_TV_AND_MOVIES'),
('v2','ENTERTAINMENT','ENTERTAINMENT_VIDEO_GAMES'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_BEER_WINE_AND_LIQUOR'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_COFFEE'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_FAST_FOOD'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_GROCERIES'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_OTHER_FOOD_AND_DRINK'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_RESTAURANT'),
('v2','FOOD_AND_DRINK','FOOD_AND_DRINK_VENDING_MACHINES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_BOOKSTORES_AND_NEWSSTANDS'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_CLOTHING_AND_ACCESSORIES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_CONVENIENCE_STORES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_DEPARTMENT_STORES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_DISCOUNT_STORES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_ELECTRONICS'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_GIFTS_AND_NOVELTIES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_OFFICE_SUPPLIES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_ONLINE_MARKETPLACES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_OTHER_GENERAL_MERCHANDISE'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_PET_SUPPLIES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_SPORTING_GOODS'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_SUPERSTORES'),
('v2','GENERAL_MERCHANDISE','GENERAL_MERCHANDISE_TOBACCO_AND_VAPE'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_ACCOUNTING_AND_FINANCIAL_PLANNING'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_AUTOMOTIVE'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_CHILDCARE'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_CONSULTING_AND_LEGAL'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_EDUCATION'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_INSURANCE'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_OTHER_GENERAL_SERVICES'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_POSTAGE_AND_SHIPPING'),
('v2','GENERAL_SERVICES','GENERAL_SERVICES_STORAGE'),
('v2','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_DONATIONS'),
('v2','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_GOVERNMENT_DEPARTMENTS_AND_AGENCIES'),
('v2','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_OTHER_GOVERNMENT_AND_NON_PROFIT'),
('v2','GOVERNMENT_AND_NON_PROFIT','GOVERNMENT_AND_NON_PROFIT_TAX_PAYMENT'),
('v2','HOME_IMPROVEMENT','HOME_IMPROVEMENT_FURNITURE'),
('v2','HOME_IMPROVEMENT','HOME_IMPROVEMENT_HARDWARE'),
('v2','HOME_IMPROVEMENT','HOME_IMPROVEMENT_OTHER_HOME_IMPROVEMENT'),
('v2','HOME_IMPROVEMENT','HOME_IMPROVEMENT_REPAIR_AND_MAINTENANCE'),
('v2','HOME_IMPROVEMENT','HOME_IMPROVEMENT_SECURITY'),
('v2','INCOME','INCOME_CHILD_SUPPORT'),
('v2','INCOME','INCOME_CONTRACTOR'),
('v2','INCOME','INCOME_DIVIDENDS'),
('v2','INCOME','INCOME_GIG_ECONOMY'),
('v2','INCOME','INCOME_INTEREST_EARNED'),
('v2','INCOME','INCOME_LONG_TERM_DISABILITY'),
('v2','INCOME','INCOME_MILITARY'),
('v2','INCOME','INCOME_OTHER'),
('v2','INCOME','INCOME_RENTAL'),
('v2','INCOME','INCOME_RETIREMENT_PENSION'),
('v2','INCOME','INCOME_SALARY'),
('v2','INCOME','INCOME_TAX_REFUND'),
('v2','INCOME','INCOME_UNEMPLOYMENT'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_AUTO'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_CASH_ADVANCES'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_EWA'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_MORTGAGE'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_OTHER_DISBURSEMENT'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_PERSONAL'),
('v2','LOAN_DISBURSEMENTS','LOAN_DISBURSEMENTS_STUDENT'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_BNPL'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_CAR_PAYMENT'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_CASH_ADVANCES'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_CREDIT_CARD_PAYMENT'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_EWA'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_MORTGAGE_PAYMENT'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_OTHER_PAYMENT'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_PERSONAL_LOAN_PAYMENT'),
('v2','LOAN_PAYMENTS','LOAN_PAYMENTS_STUDENT_LOAN_PAYMENT'),
('v2','MEDICAL','MEDICAL_DENTAL_CARE'),
('v2','MEDICAL','MEDICAL_EYE_CARE'),
('v2','MEDICAL','MEDICAL_NURSING_CARE'),
('v2','MEDICAL','MEDICAL_OTHER_MEDICAL'),
('v2','MEDICAL','MEDICAL_PHARMACIES_AND_SUPPLEMENTS'),
('v2','MEDICAL','MEDICAL_PRIMARY_CARE'),
('v2','MEDICAL','MEDICAL_VETERINARY_SERVICES'),
('v2','OTHER','OTHER_OTHER'),
('v2','PERSONAL_CARE','PERSONAL_CARE_GYMS_AND_FITNESS_CENTERS'),
('v2','PERSONAL_CARE','PERSONAL_CARE_HAIR_AND_BEAUTY'),
('v2','PERSONAL_CARE','PERSONAL_CARE_LAUNDRY_AND_DRY_CLEANING'),
('v2','PERSONAL_CARE','PERSONAL_CARE_OTHER_PERSONAL_CARE'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_GAS_AND_ELECTRICITY'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_INTERNET_AND_CABLE'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_OTHER_UTILITIES'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_RENT'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_SEWAGE_AND_WASTE_MANAGEMENT'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_TELEPHONE'),
('v2','RENT_AND_UTILITIES','RENT_AND_UTILITIES_WATER'),
('v2','TRANSFER_IN','TRANSFER_IN_ACCOUNT_TRANSFER'),
('v2','TRANSFER_IN','TRANSFER_IN_DEPOSIT'),
('v2','TRANSFER_IN','TRANSFER_IN_INVESTMENT_AND_RETIREMENT_FUNDS'),
('v2','TRANSFER_IN','TRANSFER_IN_OTHER_TRANSFER_IN'),
('v2','TRANSFER_IN','TRANSFER_IN_SAVINGS'),
('v2','TRANSFER_IN','TRANSFER_IN_TRANSFER_IN_FROM_APPS'),
('v2','TRANSFER_IN','TRANSFER_IN_WIRE'),
('v2','TRANSFER_OUT','TRANSFER_OUT_ACCOUNT_TRANSFER'),
('v2','TRANSFER_OUT','TRANSFER_OUT_CRYPTO'),
('v2','TRANSFER_OUT','TRANSFER_OUT_INVESTMENT_AND_RETIREMENT_FUNDS'),
('v2','TRANSFER_OUT','TRANSFER_OUT_OTHER_TRANSFER_OUT'),
('v2','TRANSFER_OUT','TRANSFER_OUT_SAVINGS'),
('v2','TRANSFER_OUT','TRANSFER_OUT_TRANSFER_OUT_FROM_APPS'),
('v2','TRANSFER_OUT','TRANSFER_OUT_WIRE'),
('v2','TRANSFER_OUT','TRANSFER_OUT_WITHDRAWAL'),
('v2','TRANSPORTATION','TRANSPORTATION_BIKES_AND_SCOOTERS'),
('v2','TRANSPORTATION','TRANSPORTATION_GAS'),
('v2','TRANSPORTATION','TRANSPORTATION_OTHER_TRANSPORTATION'),
('v2','TRANSPORTATION','TRANSPORTATION_PARKING'),
('v2','TRANSPORTATION','TRANSPORTATION_PUBLIC_TRANSIT'),
('v2','TRANSPORTATION','TRANSPORTATION_TAXIS_AND_RIDE_SHARES'),
('v2','TRANSPORTATION','TRANSPORTATION_TOLLS'),
('v2','TRAVEL','TRAVEL_FLIGHTS'),
('v2','TRAVEL','TRAVEL_LODGING'),
('v2','TRAVEL','TRAVEL_OTHER_TRAVEL'),
('v2','TRAVEL','TRAVEL_RENTAL_CARS');

-- Category evidence is independent of financial material equality and observation revisions.
CREATE TABLE integration.sync_category_page(
 entity_id uuid NOT NULL,book_id uuid NOT NULL,set_id uuid NOT NULL,page_index integer NOT NULL,
 body jsonb NOT NULL CHECK(jsonb_typeof(body)='object'),wire_bytes integer NOT NULL CHECK(wire_bytes BETWEEN 1 AND 2097152),
 PRIMARY KEY(entity_id,book_id,set_id,page_index),
 FOREIGN KEY(entity_id,book_id,set_id,page_index)REFERENCES integration.sync_page(entity_id,book_id,set_id,page_index)
);
CREATE TABLE banking.provider_category_revision(
 entity_id uuid NOT NULL,book_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,
 metadata_revision bigint NOT NULL CHECK(metadata_revision>0),observation_revision bigint NOT NULL CHECK(observation_revision>0),
 set_id uuid NOT NULL,page_index integer NOT NULL,category jsonb NOT NULL CHECK(jsonb_typeof(category)='object'),
 source_page_sha256 text NOT NULL CHECK(source_page_sha256 COLLATE "C" ~ '^[0-9a-f]{64}$'),
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),
 PRIMARY KEY(entity_id,book_id,observation_id,metadata_revision),UNIQUE(entity_id,book_id,set_id,observation_id),
 FOREIGN KEY(entity_id,book_id,observation_id,observation_revision)REFERENCES banking.observation_revision(entity_id,book_id,observation_id,revision),
 FOREIGN KEY(entity_id,book_id,set_id,page_index)REFERENCES integration.sync_category_page(entity_id,book_id,set_id,page_index)
);
CREATE TABLE integration.sync_category_completion(
 entity_id uuid NOT NULL,book_id uuid NOT NULL,set_id uuid NOT NULL,result jsonb NOT NULL,
 PRIMARY KEY(entity_id,book_id,set_id),FOREIGN KEY(entity_id,book_id,set_id)REFERENCES integration.sync_completion
);
CREATE INDEX provider_category_current ON banking.provider_category_revision(entity_id,book_id,observation_id,metadata_revision DESC);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_category_page FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.provider_category_revision FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON integration.sync_category_completion FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();

-- Recompute safe unavailable reasons rather than trusting a caller's confidence/eligibility flag.
CREATE FUNCTION banking.automatic_category_metadata(v jsonb)RETURNS jsonb LANGUAGE plpgsql IMMUTABLE SET search_path=pg_catalog,pg_temp AS $$
DECLARE presence_value text;reason_value text;code_value text;version_value text;confidence_value text;k text;BEGIN
 PERFORM banking.collection_keys(v,ARRAY['presence','version','primary','detailed','confidence','unavailable_reason']);
 presence_value:=ledger.text_value(v->'presence');
 IF presence_value IN('omitted','null')THEN
  IF EXISTS(SELECT FROM unnest(ARRAY['version','primary','detailed','confidence'])x WHERE v->x<>'null'::jsonb)THEN PERFORM banking.collection_error('P2800');END IF;
  reason_value:=CASE WHEN presence_value='omitted'THEN 'not_provided'ELSE 'null_category'END;
 ELSIF presence_value='object'THEN
  FOREACH k IN ARRAY ARRAY['primary','detailed']LOOP
   IF v->k<>'null'::jsonb THEN code_value:=ledger.text_value(v->k);IF length(code_value)>128 OR code_value COLLATE "C" !~ '^[A-Z][A-Z0-9_]*$'THEN PERFORM banking.collection_error('P2800');END IF;END IF;
  END LOOP;
  IF v->'version'<>'null'::jsonb THEN version_value:=ledger.text_value(v->'version');IF length(version_value)>16 OR version_value COLLATE "C" !~ '^[A-Za-z0-9][A-Za-z0-9._-]*$'THEN PERFORM banking.collection_error('P2800');END IF;END IF;
  IF v->'confidence'<>'null'::jsonb THEN confidence_value:=ledger.text_value(v->'confidence');IF length(confidence_value)>32 OR confidence_value COLLATE "C" !~ '^[A-Z][A-Z0-9_]*$'THEN PERFORM banking.collection_error('P2800');END IF;END IF;
  reason_value:=CASE WHEN v->'primary'='null'::jsonb OR v->'detailed'='null'::jsonb THEN 'incomplete_category'
   WHEN version_value IS NULL THEN 'version_missing'WHEN version_value NOT IN('v1','v2')THEN 'unsupported_version'
   WHEN confidence_value IS NULL THEN 'confidence_missing'WHEN confidence_value='UNKNOWN'THEN 'confidence_unknown'
   WHEN confidence_value NOT IN('VERY_HIGH','HIGH','MEDIUM','LOW')THEN 'confidence_unrecognized'ELSE ''END;
 ELSE PERFORM banking.collection_error('P2800');END IF;
 IF ledger.text_value(v->'unavailable_reason')<>reason_value THEN PERFORM banking.collection_error('P2800');END IF;
 RETURN v;
END $$;

CREATE FUNCTION integration.stage_sync_category_page(e uuid,b uuid,id uuid,set_value uuid,f bigint,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;c jsonb;a jsonb;old integration.sync_category_page;res jsonb;idx integer;seen jsonb:='[]';kind_value text;total_bytes bigint;BEGIN
 v:=banking.import_json(raw,2097152);c:=v->'category_attribution';
 PERFORM banking.collection_keys(c,ARRAY['profile','source_page_sha256','attributions']);
 IF c->>'profile'IS DISTINCT FROM 'jayburd-plaid-pfc-attribution/1'OR c->>'source_page_sha256'IS DISTINCT FROM v->>'sha256'
  OR jsonb_typeof(c->'attributions')IS DISTINCT FROM 'array'OR jsonb_array_length(c->'attributions')>500 THEN PERFORM integration.sync_error('P2600');END IF;
 -- Existing worker authorization, attempt, identity/money and cursor fences run first in this transaction.
 res:=integration.stage_sync_page(e,b,id,set_value,f,(v-'category_attribution')::text);idx:=(v->>'page_index')::integer;
 SELECT * INTO old FROM integration.sync_category_page WHERE entity_id=e AND book_id=b AND set_id=set_value AND page_index=idx;
 IF FOUND THEN IF old.body<>c OR old.wire_bytes<>octet_length(raw)THEN PERFORM integration.sync_error('P2602');END IF;RETURN res;END IF;
 FOR a IN SELECT value FROM jsonb_array_elements(c->'attributions')LOOP
  PERFORM banking.collection_keys(a,ARRAY['account_id','transaction_id','change','category']);kind_value:=ledger.text_value(a->'change');
  IF kind_value NOT IN('added','modified')OR NOT EXISTS(SELECT FROM jsonb_array_elements(v->kind_value)t WHERE t->>'account_id'=a->>'account_id'AND t->>'transaction_id'=a->>'transaction_id')
   OR EXISTS(SELECT FROM jsonb_array_elements(seen)x WHERE x->>'account_id'=a->>'account_id'AND x->>'transaction_id'=a->>'transaction_id')THEN PERFORM integration.sync_error('P2600');END IF;
  PERFORM banking.automatic_category_metadata(a->'category');seen:=seen||jsonb_build_array(a);
 END LOOP;
 IF jsonb_array_length(seen)<>jsonb_array_length(v->'added')+jsonb_array_length(v->'modified')THEN PERFORM integration.sync_error('P2600');END IF;
 SELECT coalesce(sum(wire_bytes),0)+octet_length(raw)INTO total_bytes FROM integration.sync_category_page WHERE entity_id=e AND book_id=b AND set_id=set_value;
 IF total_bytes>2097152 OR (SELECT count(*)FROM integration.sync_page WHERE entity_id=e AND book_id=b AND set_id=set_value)<>(SELECT count(*)+1 FROM integration.sync_category_page WHERE entity_id=e AND book_id=b AND set_id=set_value)THEN PERFORM integration.sync_error('P2605');END IF;
 INSERT INTO integration.sync_category_page VALUES(e,b,set_value,idx,c,octet_length(raw));RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR SQLSTATE 'P2800'OR invalid_text_representation THEN PERFORM integration.sync_error('P2600');
END $$;

CREATE FUNCTION integration.sync_category_context(e uuid,b uuid,id uuid,set_value uuid,f bigint)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$DECLARE res jsonb;n bigint;bytes_value bigint;BEGIN
 res:=integration.sync_context(e,b,id,set_value,f);
 SELECT count(*),coalesce(sum(wire_bytes),0)INTO n,bytes_value FROM integration.sync_category_page WHERE entity_id=e AND book_id=b AND set_id=set_value;
 IF n<>(res->>'page_count')::bigint OR bytes_value>2097152 THEN PERFORM integration.sync_error('P2602');END IF;
 RETURN jsonb_set(res,'{byte_count}',to_jsonb(bytes_value));
END $$;

CREATE FUNCTION integration.complete_sync_categories(e uuid,b uuid,id uuid,set_value uuid,f bigint)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE res jsonb;t integration.sync_stream;p record;a jsonb;o banking.source_observation;old banking.provider_category_revision;next_value bigint;w uuid;changed boolean:=false;BEGIN
 -- Current worker scope and replay are checked even when a retained completion exists.
 t:=integration.sync_authorize(e,b,id);
 IF (SELECT count(*)FROM integration.sync_page WHERE entity_id=e AND book_id=b AND set_id=set_value)<>(SELECT count(*)FROM integration.sync_category_page WHERE entity_id=e AND book_id=b AND set_id=set_value)THEN PERFORM integration.sync_error('P2602');END IF;
 res:=integration.complete_sync(e,b,id,set_value,f);
 IF EXISTS(SELECT FROM integration.sync_category_completion WHERE entity_id=e AND book_id=b AND set_id=set_value)THEN RETURN res;END IF;
 FOR p IN SELECT * FROM integration.sync_category_page WHERE entity_id=e AND book_id=b AND set_id=set_value ORDER BY page_index LOOP
  FOR a IN SELECT value FROM jsonb_array_elements(p.body->'attributions')LOOP
   SELECT * INTO STRICT o FROM banking.source_observation WHERE entity_id=e AND book_id=b AND source_account_id=t.source_account_id AND namespace_id=t.namespace_id AND external_entry_id=a->>'transaction_id';
   IF EXISTS(SELECT FROM banking.provider_category_revision WHERE entity_id=e AND book_id=b AND set_id=set_value AND observation_id=o.observation_id)THEN CONTINUE;END IF;
   SELECT * INTO old FROM banking.provider_category_revision WHERE entity_id=e AND book_id=b AND observation_id=o.observation_id ORDER BY metadata_revision DESC LIMIT 1;
   IF FOUND AND old.observation_revision=o.current_revision AND old.category=a->'category'THEN CONTINUE;END IF;
   next_value:=coalesce(old.metadata_revision,0)+1;
   INSERT INTO banking.provider_category_revision(entity_id,book_id,source_account_id,observation_id,metadata_revision,observation_revision,set_id,page_index,category,source_page_sha256)
    VALUES(e,b,t.source_account_id,o.observation_id,next_value,o.current_revision,set_value,p.page_index,a->'category',p.body->>'source_page_sha256');changed:=true;
  END LOOP;
 END LOOP;
 IF changed THEN
  FOR w IN SELECT DISTINCT workspace_id FROM banking.collection_source WHERE origin='legacy'AND legacy_entity_id=e AND legacy_book_id=b AND legacy_source_account_id=t.source_account_id ORDER BY workspace_id LOOP
   UPDATE banking.collection_label_state SET projection_revision=projection_revision+1 WHERE workspace_id=w;
  END LOOP;
 END IF;
 INSERT INTO integration.sync_category_completion VALUES(e,b,set_value,res);RETURN res;
END $$;

-- AUTOMATION-POLICY-AND-PROJECTION
CREATE TABLE banking.collection_automation_edition(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,revision bigint NOT NULL CHECK(revision>=0),
 enabled boolean NOT NULL,profile text NOT NULL DEFAULT 'automatic-analytic-policy-v1'CHECK(profile='automatic-analytic-policy-v1'),
 map_edition text NOT NULL DEFAULT 'plaid-pfc-primary-2026-11-11',principal_id uuid NOT NULL REFERENCES ledger.principal,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),PRIMARY KEY(workspace_id,revision)
);
CREATE TABLE banking.collection_automation_state(
 workspace_id uuid PRIMARY KEY REFERENCES banking.collection_workspace,revision bigint NOT NULL DEFAULT 0,
 initialized boolean NOT NULL DEFAULT false,last_checked_at timestamptz,last_run_id uuid,
 blocked_reason text CHECK(blocked_reason IN('saved_work','category_catalog_unavailable','permission_changed')),
 FOREIGN KEY(workspace_id,revision)REFERENCES banking.collection_automation_edition DEFERRABLE INITIALLY DEFERRED,
 CHECK(last_checked_at IS NULL OR isfinite(last_checked_at))
);
CREATE TABLE banking.collection_automation_setting_receipt(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,principal_id uuid NOT NULL REFERENCES ledger.principal,
 request_key text NOT NULL CHECK(request_key COLLATE "C" ~ '^[A-Za-z0-9._:-]{1,128}$'),request jsonb NOT NULL,result jsonb NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),PRIMARY KEY(workspace_id,principal_id,request_key),
 CHECK(octet_length((request::text||result::text))<=16384)
);
CREATE TABLE banking.collection_automatic_run(
 workspace_id uuid NOT NULL REFERENCES banking.collection_workspace,run_id uuid NOT NULL DEFAULT gen_random_uuid(),
 work_key text NOT NULL CHECK(work_key COLLATE "C" ~ '^[0-9a-f]{64}$'),principal_id uuid NOT NULL REFERENCES ledger.principal,
 policy_revision bigint NOT NULL,context jsonb NOT NULL,evaluations jsonb NOT NULL,input_fingerprints jsonb NOT NULL,
 state text NOT NULL DEFAULT 'prepared'CHECK(state IN('prepared','committed','stale')),result jsonb,
 created_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(created_at)),terminal_at timestamptz,
 PRIMARY KEY(workspace_id,run_id),UNIQUE(workspace_id,work_key),
 FOREIGN KEY(workspace_id,policy_revision)REFERENCES banking.collection_automation_edition,
 CHECK(octet_length((context::text||evaluations::text||input_fingerprints::text))<=2097152),
 CHECK((state='prepared'AND result IS NULL AND terminal_at IS NULL)OR(state<>'prepared'AND result IS NOT NULL AND terminal_at IS NOT NULL AND isfinite(terminal_at)))
);
CREATE UNIQUE INDEX collection_automatic_one_prepared ON banking.collection_automatic_run(workspace_id)WHERE state='prepared';
ALTER TABLE banking.collection_automation_state ADD FOREIGN KEY(workspace_id,last_run_id)REFERENCES banking.collection_automatic_run DEFERRABLE INITIALLY DEFERRED;
CREATE TABLE banking.collection_automatic_decision(
 workspace_id uuid NOT NULL,source_account_id uuid NOT NULL,observation_id uuid NOT NULL,observation_revision bigint NOT NULL CHECK(observation_revision>0),
 decision_revision bigint NOT NULL CHECK(decision_revision>0),run_id uuid NOT NULL,input_fingerprint text NOT NULL CHECK(input_fingerprint COLLATE "C" ~ '^[0-9a-f]{64}$'),
 output jsonb,classification jsonb NOT NULL,rules jsonb NOT NULL,source_attribution jsonb,policy_revision bigint NOT NULL,
 recorded_at timestamptz NOT NULL DEFAULT clock_timestamp()CHECK(isfinite(recorded_at)),
 PRIMARY KEY(workspace_id,decision_revision,source_account_id,observation_id),
 FOREIGN KEY(workspace_id,source_account_id)REFERENCES banking.collection_source,
 FOREIGN KEY(workspace_id,run_id)REFERENCES banking.collection_automatic_run,
 FOREIGN KEY(workspace_id,policy_revision)REFERENCES banking.collection_automation_edition,
 CHECK(octet_length((coalesce(output::text,'')||classification::text||rules::text||coalesce(source_attribution::text,'')))<=131072)
);
CREATE INDEX collection_automatic_current ON banking.collection_automatic_decision(workspace_id,source_account_id,observation_id,decision_revision DESC);
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_automation_edition FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_automation_setting_receipt FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE TRIGGER immutable BEFORE UPDATE OR DELETE ON banking.collection_automatic_decision FOR EACH ROW EXECUTE FUNCTION ledger.immutable_row();
CREATE FUNCTION banking.collection_automatic_run_guard()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 IF TG_OP='DELETE'OR OLD.state<>'prepared'OR NEW.state NOT IN('committed','stale')OR(to_jsonb(NEW)-ARRAY['state','result','terminal_at'])<>(to_jsonb(OLD)-ARRAY['state','result','terminal_at'])THEN PERFORM banking.collection_error('P2802');END IF;RETURN NEW;
END $$;
CREATE TRIGGER preserve_automatic_run BEFORE UPDATE OR DELETE ON banking.collection_automatic_run FOR EACH ROW EXECUTE FUNCTION banking.collection_automatic_run_guard();
-- Existing workspaces never acquire an automatic mandate merely from installing a migration.
INSERT INTO banking.collection_automation_edition(workspace_id,revision,enabled,principal_id)SELECT workspace_id,0,false,owner_principal_id FROM banking.collection_workspace;
INSERT INTO banking.collection_automation_state(workspace_id)SELECT workspace_id FROM banking.collection_workspace;
CREATE FUNCTION banking.collection_automatic_initialize()RETURNS trigger LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$BEGIN
 INSERT INTO banking.collection_automation_edition(workspace_id,revision,enabled,principal_id)VALUES(NEW.workspace_id,0,true,NEW.owner_principal_id);
 INSERT INTO banking.collection_automation_state(workspace_id)VALUES(NEW.workspace_id);RETURN NEW;
END $$;
CREATE TRIGGER collection_automatic_initialize AFTER INSERT ON banking.collection_workspace FOR EACH ROW EXECUTE FUNCTION banking.collection_automatic_initialize();

CREATE FUNCTION banking.collection_automatic_render(r banking.collection_automatic_run)RETURNS jsonb LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT coalesce(r.result,jsonb_build_object('profile','private-collection-auto-run-v1','workspace_id',r.workspace_id,'run_id',r.run_id,'state',r.state,'processed_count','0','attention_count','0'))
$$;

-- Starter installation is a bounded, current-authorized worker mutation, never a read effect.
CREATE FUNCTION banking.collection_automatic_starter(actor uuid,w uuid)RETURNS boolean LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE st banking.collection_label_state;topics_value jsonb;expected jsonb;parent_id uuid:='a4880000-0000-4000-8000-000000000000';x jsonb;existing jsonb;BEGIN
 IF(SELECT initialized FROM banking.collection_automation_state WHERE workspace_id=w)THEN RETURN true;END IF;
 SELECT * INTO STRICT st FROM banking.collection_label_state WHERE workspace_id=w;
 SELECT topics INTO STRICT topics_value FROM banking.collection_taxonomy_edition WHERE workspace_id=w AND taxonomy_revision=st.taxonomy_revision;
 SELECT jsonb_build_array(jsonb_build_object('topic_id',parent_id,'name','Automatic categories','parent_id',NULL,'status','active'))||
  coalesce(jsonb_agg(jsonb_build_object('topic_id',topic_id,'name',name,'parent_id',parent_id,'status','active')ORDER BY code),'[]')INTO expected FROM reference.analytic_category;
 FOR x IN SELECT value FROM jsonb_array_elements(expected)LOOP
  SELECT value INTO existing FROM jsonb_array_elements(topics_value)WHERE value->>'topic_id'=x->>'topic_id';
  IF FOUND THEN IF existing<>x THEN RETURN false;END IF;
  ELSE
   IF EXISTS(SELECT FROM jsonb_array_elements(topics_value)t WHERE t->'parent_id'=x->'parent_id'AND t->>'name'=x->>'name')THEN RETURN false;END IF;
   topics_value:=topics_value||jsonb_build_array(x);
  END IF;
 END LOOP;
 IF jsonb_array_length(topics_value)>1000 OR(SELECT count(*)FROM jsonb_array_elements(topics_value)t WHERE t->>'status'='active')>500 OR octet_length(topics_value::text)>524288 THEN RETURN false;END IF;
 IF topics_value<>(SELECT topics FROM banking.collection_taxonomy_edition WHERE workspace_id=w AND taxonomy_revision=st.taxonomy_revision)THEN
  INSERT INTO banking.collection_taxonomy_edition(workspace_id,taxonomy_revision,topics,principal_id)VALUES(w,st.taxonomy_revision+1,topics_value,actor);
  UPDATE banking.collection_label_state SET taxonomy_revision=taxonomy_revision+1 WHERE workspace_id=w;
 END IF;
 UPDATE banking.collection_automation_state SET initialized=true,blocked_reason=NULL WHERE workspace_id=w;RETURN true;
END $$;

-- This private helper reads only current explicitly adopted source attribution; no historical raw scan.
CREATE FUNCTION banking.collection_automatic_provider(w uuid,sid uuid,oid uuid,rev bigint)RETURNS jsonb LANGUAGE sql STABLE AS $$
 SELECT jsonb_build_object('metadata_revision',p.metadata_revision::text,'category',p.category,'source_page_sha256',p.source_page_sha256)
 FROM banking.collection_source s JOIN banking.provider_category_revision p ON s.origin='legacy'AND p.entity_id=s.legacy_entity_id AND p.book_id=s.legacy_book_id AND p.source_account_id=s.legacy_source_account_id AND p.observation_id=oid AND p.observation_revision=rev
 WHERE s.workspace_id=w AND s.source_account_id=sid ORDER BY p.metadata_revision DESC LIMIT 1
$$;

-- SQL-only helpers inherit the secure entrypoint search_path; fully qualified tables permit planner inlining.
-- Do not include global auto progress in row fingerprints: one batch must not invalidate every earlier batch.
CREATE FUNCTION banking.collection_automatic_row_fingerprint(w uuid,sid uuid,oid uuid,rev bigint,body_value jsonb)RETURNS text LANGUAGE sql STABLE AS $$
 SELECT encode(sha256(convert_to(jsonb_build_array(rev::text,body_value->'source_state',body_value->'money',
  s.taxonomy_revision::text,s.tag_revision::text,s.ruleset_revision::text,
  (SELECT max(d.decision_revision)::text FROM banking.collection_label_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid),
  (SELECT max(d.decision_revision)::text FROM banking.collection_tag_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid),
  (SELECT max(d.decision_revision)::text FROM banking.collection_rule_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid),
  banking.collection_automatic_provider(w,sid,oid,rev),'automatic-analytic-policy-v1','plaid-pfc-primary-2026-11-11')::text,'UTF8')),'hex')
 FROM banking.collection_label_state s WHERE s.workspace_id=w
$$;

ALTER FUNCTION banking.collection_query_rows(uuid,uuid)RENAME TO collection_automatic_previous_rows;
CREATE FUNCTION banking.collection_query_rows(actor uuid,w uuid)RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,booked_date date,body jsonb)
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 WITH current_rows AS MATERIALIZED(
  SELECT r.*,CASE WHEN EXISTS(SELECT FROM banking.collection_automatic_decision d WHERE d.workspace_id=w AND d.source_account_id=r.source_id AND d.observation_id=r.observation_id)THEN banking.collection_automatic_row_fingerprint(w,r.source_id,r.observation_id,r.observation_revision,r.body)END AS input_fingerprint
  FROM banking.collection_automatic_previous_rows(actor,w)r
 )
 SELECT r.source_id,r.observation_id,r.observation_revision,r.booked_date,
  r.body||jsonb_build_object('owner_topic',CASE WHEN manual_topic.present THEN r.body->'owner_topic'
   WHEN d.input_fingerprint=r.input_fingerprint THEN CASE WHEN d.output IS NULL THEN jsonb_build_object('state','unclassified')WHEN d.output->>'kind'='uncategorized'THEN jsonb_build_object('state','uncategorized')ELSE jsonb_build_object('state','topic','topic_id',d.output->>'topic_id','name',topic.value->>'name')END
   ELSE r.body->'owner_topic'END,
  'owner_tags',CASE WHEN NOT manual_tags.present AND d.classification->>'origin'='rule'AND d.input_fingerprint=r.input_fingerprint AND d.output IS NOT NULL THEN jsonb_build_object('state','current','tags',banking.collection_tag_names(w,ARRAY(SELECT value::uuid FROM jsonb_array_elements_text(d.output->'tag_ids'))))ELSE r.body->'owner_tags'END,
  'categorization',CASE WHEN manual_topic.present THEN jsonb_build_object('state',CASE WHEN r.body->'owner_topic'->>'state'='stale'THEN 'needs_attention'ELSE 'manual'END,'origin','manual','reason',CASE WHEN r.body->'owner_topic'->>'state'='stale'THEN 'stale_override'ELSE NULL END,'rule_names','[]'::jsonb,'provider',NULL,'decision_id',NULL)
   WHEN d.input_fingerprint=r.input_fingerprint THEN d.classification||jsonb_build_object('decision_id',d.run_id)
   ELSE jsonb_build_object('state',CASE WHEN r.body->>'source_state' IN('pending','removed','held')THEN 'unavailable'WHEN policy.enabled THEN 'pending'ELSE 'unavailable'END,'origin','none','reason',CASE WHEN r.body->>'source_state'<>'booked'THEN 'source_not_ready'ELSE 'unavailable'END,'rule_names','[]'::jsonb,'provider',NULL,'decision_id',NULL)END)
 FROM current_rows r
 JOIN banking.collection_automation_state a ON a.workspace_id=w JOIN banking.collection_automation_edition policy ON policy.workspace_id=w AND policy.revision=a.revision
 LEFT JOIN LATERAL(SELECT x.* FROM banking.collection_automatic_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id ORDER BY decision_revision DESC LIMIT 1)d ON true
 CROSS JOIN LATERAL(SELECT EXISTS(SELECT FROM banking.collection_label_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id)present)manual_topic
 CROSS JOIN LATERAL(SELECT EXISTS(SELECT FROM banking.collection_tag_decision x WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id)present)manual_tags
 LEFT JOIN LATERAL(SELECT value FROM banking.collection_taxonomy_edition t JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(t.topics)x(value)WHERE t.workspace_id=w AND value->>'topic_id'=d.output->>'topic_id')topic ON true
$$;

-- Exact exceptions filter uses the complete persisted projection before count and pagination.
ALTER FUNCTION banking.query_collection_review_v3(uuid,uuid,text)RENAME TO collection_automatic_previous_query;
CREATE FUNCTION banking.query_collection_review_v3(actor uuid,w uuid,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;sel jsonb;snap jsonb;sid uuid;first_day date;last_day date;search_value text;assignment_value text;
 topic_kind text;tid uuid;tag_kind text;tagid uuid;merchant_kind text;merchant_value text;institution_kind text;owner_kind text;
 amount_currency uuid;amount_min numeric;amount_max numeric;amount_direction text;units integer;
 book_kind text;eid uuid;bid uuid;source_status text;journal_status text;sort_field text;sort_order text;
 cur jsonb;txt text;page_offset bigint:=0;res jsonb;total bigint;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);v:=banking.import_json(raw,8192);
 PERFORM banking.collection_keys(v,ARRAY['profile','source_account_id','assignment','starts_on','through_date','search','topic','tag','merchant','amount','institution','legal_owner','book','status','sort','snapshot','cursor','limit']);
 IF ledger.text_value(v->'profile')<>'private-collection-review-v3'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_typeof(v->'limit')IS DISTINCT FROM 'number'OR v->>'limit'!~ '^[1-9][0-9]{0,2}$'OR(v->>'limit')::integer>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'source_account_id'<>'null'::jsonb THEN sid:=ledger.uuid_value(v->'source_account_id');IF NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)WHERE source_account_id=sid)THEN PERFORM banking.collection_error('P2801');END IF;END IF;
 assignment_value:=ledger.text_value(v->'assignment');search_value:=ledger.text_value(v->'search');
 IF assignment_value<>ALL(ARRAY['all','assigned','unassigned'])OR length(search_value)>100 THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'starts_on'<>'null'::jsonb THEN first_day:=ledger.date_value(ledger.text_value(v->'starts_on'));END IF;
 IF v->'through_date'<>'null'::jsonb THEN last_day:=ledger.date_value(ledger.text_value(v->'through_date'));END IF;
 IF first_day>last_day THEN PERFORM banking.collection_error('P2800');END IF;
 topic_kind:=ledger.text_value(v->'topic'->'kind');
 IF topic_kind='topic'THEN PERFORM banking.collection_keys(v->'topic',ARRAY['kind','topic_id']);tid:=ledger.uuid_value(v->'topic'->'topic_id');
  IF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'topic_id'=tid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'topic',ARRAY['kind']);IF topic_kind<>ALL(ARRAY['all','unclassified','uncategorized','needs_attention'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 tag_kind:=ledger.text_value(v->'tag'->'kind');
 IF tag_kind='tag'THEN PERFORM banking.collection_keys(v->'tag',ARRAY['kind','tag_id']);tagid:=ledger.uuid_value(v->'tag'->'tag_id');
  IF NOT EXISTS(SELECT FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'tag_id'=tagid::text)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'tag',ARRAY['kind']);IF tag_kind<>ALL(ARRAY['all','untagged'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 merchant_kind:=ledger.text_value(v->'merchant'->'kind');
 IF merchant_kind IN('exact','contains')THEN PERFORM banking.collection_keys(v->'merchant',ARRAY['kind','value']);merchant_value:=ledger.text_value(v->'merchant'->'value');IF length(merchant_value)NOT BETWEEN 1 AND 100 THEN PERFORM banking.collection_error('P2800');END IF;
 ELSE PERFORM banking.collection_keys(v->'merchant',ARRAY['kind']);IF merchant_kind<>ALL(ARRAY['all','missing'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 PERFORM banking.collection_keys(v->'institution',ARRAY['kind']);institution_kind:=ledger.text_value(v->'institution'->'kind');IF institution_kind<>ALL(ARRAY['all','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_keys(v->'legal_owner',ARRAY['kind']);owner_kind:=ledger.text_value(v->'legal_owner'->'kind');IF owner_kind<>ALL(ARRAY['all','unknown'])THEN PERFORM banking.collection_error('P2800');END IF;
 book_kind:=ledger.text_value(v->'book'->'kind');
 IF book_kind='book'THEN PERFORM banking.collection_keys(v->'book',ARRAY['kind','entity_id','book_id']);eid:=ledger.uuid_value(v->'book'->'entity_id');bid:=ledger.uuid_value(v->'book'->'book_id');
  IF NOT banking.collection_label_book_read(actor,eid,bid)OR NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)s WHERE s.origin='legacy'AND s.legacy_entity_id=eid AND s.legacy_book_id=bid)THEN PERFORM banking.collection_error('P2801');END IF;
 ELSE PERFORM banking.collection_keys(v->'book',ARRAY['kind']);IF book_kind<>ALL(ARRAY['all','unassigned','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;END IF;
 PERFORM banking.collection_keys(v->'status',ARRAY['source','journal']);source_status:=ledger.text_value(v->'status'->'source');journal_status:=ledger.text_value(v->'status'->'journal');
 IF source_status<>ALL(ARRAY['all','booked','pending','removed','held'])OR journal_status<>ALL(ARRAY['all','unposted','posted','correction_required','unavailable'])THEN PERFORM banking.collection_error('P2800');END IF;
 IF v->'amount'<>'null'::jsonb THEN
  PERFORM banking.collection_keys(v->'amount',ARRAY['currency_version_id','minimum','maximum','direction']);amount_currency:=ledger.uuid_value(v->'amount'->'currency_version_id');amount_direction:=ledger.text_value(v->'amount'->'direction');
  IF amount_direction<>ALL(ARRAY['all','inflow','outflow','zero'])THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT c.minor_unit INTO units FROM reference.currency c WHERE c.currency_version_id=amount_currency;
  IF NOT FOUND OR units IS NULL THEN PERFORM banking.collection_error('P2801');END IF;
  amount_min:=banking.collection_query_amount(v->'amount'->'minimum',units);amount_max:=banking.collection_query_amount(v->'amount'->'maximum',units);
  IF amount_min>amount_max THEN PERFORM banking.collection_error('P2800');END IF;
 END IF;
 PERFORM banking.collection_keys(v->'sort',ARRAY['field','order']);sort_field:=ledger.text_value(v->'sort'->'field');sort_order:=ledger.text_value(v->'sort'->'order');
 IF sort_field<>ALL(ARRAY['date','account','description','merchant','amount','currency','book','source_state','journal_state','topic','tags','institution','legal_owner'])OR sort_order<>ALL(ARRAY['asc','desc'])THEN PERFORM banking.collection_error('P2800');END IF;
 snap:=banking.collection_query_snapshot(w);IF v->'snapshot'<>'null'::jsonb THEN PERFORM banking.collection_query_pins(v->'snapshot',snap);END IF;sel:=v-ARRAY['profile','snapshot','cursor','limit'];
 IF v->'cursor'<>'null'::jsonb THEN
  txt:=ledger.text_value(v->'cursor');IF length(txt)NOT BETWEEN 2 AND 4096 OR txt COLLATE "C" !~ '^([0-9a-f]{2})+$'THEN PERFORM banking.collection_error('P2800');END IF;
  cur:=banking.import_json(convert_from(decode(txt,'hex'),'UTF8'),2048);PERFORM banking.collection_keys(cur,ARRAY['workspace_id','snapshot','selector_hash','offset']);
  IF ledger.uuid_value(cur->'workspace_id')<>w OR ledger.text_value(cur->'selector_hash')<>encode(sha256(convert_to(sel::text,'UTF8')),'hex')THEN PERFORM banking.collection_error('P2800');END IF;
  PERFORM banking.collection_query_pins(cur->'snapshot',snap);page_offset:=banking.import_revision(cur->'offset');IF page_offset=0 THEN PERFORM banking.collection_error('P2800');END IF;
 END IF;
 WITH facts AS MATERIALIZED(SELECT r.*,s.origin,s.legacy_entity_id,s.legacy_book_id FROM banking.collection_query_rows(actor,w)r JOIN banking.collection_source s ON s.workspace_id=w AND s.source_account_id=r.source_id),
 matched AS MATERIALIZED(SELECT f.* FROM facts f WHERE(sid IS NULL OR f.source_id=sid)AND(first_day IS NULL OR f.booked_date>=first_day)AND(last_day IS NULL OR f.booked_date<=last_day)
 AND strpos(lower(f.body->>'description'COLLATE "C"),lower(search_value COLLATE "C"))>0
 AND(assignment_value='all'OR f.body->'assignment'->>'state'=assignment_value)
 AND(topic_kind='needs_attention'AND f.body->>'source_state'='booked'AND f.body->'categorization'->>'state'='needs_attention'OR topic_kind='all'OR topic_kind='unclassified'AND f.body->'owner_topic'->>'state'IN('unclassified','stale')OR topic_kind='uncategorized'AND f.body->'owner_topic'->>'state'='uncategorized'OR topic_kind='topic'AND f.body->'owner_topic'->>'state'='topic'AND f.body->'owner_topic'->>'topic_id'=tid::text)
 AND(tag_kind='all'OR tag_kind='untagged'AND(f.body->'owner_tags'->>'state'<>'current'OR jsonb_array_length(f.body->'owner_tags'->'tags')=0)OR tag_kind='tag'AND f.body->'owner_tags'->>'state'='current'AND EXISTS(SELECT FROM jsonb_array_elements(f.body->'owner_tags'->'tags')t WHERE t->>'tag_id'=tagid::text))
 AND(merchant_kind='all'OR merchant_kind='missing'AND(f.body->>'merchant'IS NULL OR f.body->>'merchant'='')OR merchant_kind='exact'AND lower(f.body->>'merchant'COLLATE "C")=lower(merchant_value COLLATE "C")OR merchant_kind='contains'AND strpos(lower(f.body->>'merchant'COLLATE "C"),lower(merchant_value COLLATE "C"))>0)
 AND(book_kind='all'OR book_kind='unassigned'AND f.body->'assignment'->>'state'='unassigned'OR book_kind='unavailable'AND f.body->'assignment'->>'state'='unavailable'OR book_kind='book'AND f.body->'assignment'->>'state'='assigned'AND f.legacy_entity_id=eid AND f.legacy_book_id=bid)
 AND(source_status='all'OR f.body->>'source_state'=source_status)AND(journal_status='all'OR f.body->>'journal_state'=journal_status)
 AND(amount_currency IS NULL OR f.body->'money'->>'state'='available'AND f.body->'money'->>'currency_version_id'=amount_currency::text AND(amount_direction='all'OR f.body->'money'->>'direction'=amount_direction)AND(amount_min IS NULL OR(f.body->'money'->>'amount')::numeric>=amount_min)AND(amount_max IS NULL OR(f.body->'money'->>'amount')::numeric<=amount_max))),
 keyed AS MATERIALIZED(SELECT f.*,
 CASE sort_field WHEN 'date'THEN to_char(booked_date,'YYYY-MM-DD')WHEN 'account'THEN body->>'account_name'WHEN 'description'THEN body->>'description'WHEN 'merchant'THEN nullif(body->>'merchant','')WHEN 'amount'THEN body->'money'->>'currency'WHEN 'currency'THEN body->'money'->>'currency'WHEN 'book'THEN body->'assignment'->'books'->0->>'name'WHEN 'source_state'THEN body->>'source_state'WHEN 'journal_state'THEN body->>'journal_state'WHEN 'topic'THEN CASE WHEN body->'owner_topic'->>'state'='topic'THEN body->'owner_topic'->>'name'WHEN body->'owner_topic'->>'state'='uncategorized'THEN 'Uncategorized'END WHEN 'tags'THEN(SELECT string_agg(t->>'name',E'\n'ORDER BY(t->>'name')COLLATE "C")FROM jsonb_array_elements(body->'owner_tags'->'tags')t)END text_key,
 CASE WHEN sort_field IN('amount','currency')THEN body->'money'->>'currency_version_id'END currency_key,
 CASE WHEN sort_field='amount'AND body->'money'->>'state'='available'THEN(body->'money'->>'amount')::numeric END amount_key FROM matched f),
 numbered AS MATERIALIZED(SELECT f.*,row_number()OVER(ORDER BY CASE WHEN sort_order='asc'THEN text_key END COLLATE "C" ASC NULLS LAST,CASE WHEN sort_order='desc'THEN text_key END COLLATE "C" DESC NULLS LAST,CASE WHEN sort_order='asc'THEN currency_key END ASC NULLS LAST,CASE WHEN sort_order='desc'THEN currency_key END DESC NULLS LAST,CASE WHEN sort_order='asc'THEN amount_key END ASC NULLS LAST,CASE WHEN sort_order='desc'THEN amount_key END DESC NULLS LAST,source_id,observation_id)rn FROM keyed f),
 picked AS MATERIALIZED(SELECT * FROM numbered WHERE rn>page_offset ORDER BY rn LIMIT(v->>'limit')::integer)
 SELECT jsonb_build_object('profile','private-collection-review-v3','workspace_id',w,'selector',sel,'snapshot',snap,'matching_count',(SELECT count(*)::text FROM matched),'start_index',CASE WHEN EXISTS(SELECT FROM picked)THEN(page_offset+1)::text ELSE '0'END,'items',coalesce((SELECT jsonb_agg(body ORDER BY rn)FROM picked),'[]'),
 'next_cursor',CASE WHEN(SELECT max(rn)FROM picked)<(SELECT count(*)FROM matched)THEN encode(convert_to(jsonb_build_object('workspace_id',w,'snapshot',snap,'selector_hash',encode(sha256(convert_to(sel::text,'UTF8')),'hex'),'offset',(SELECT max(rn)::text FROM picked))::text,'UTF8'),'hex')END,'partial',EXISTS(SELECT FROM banking.collection_visible_sources(w)coverage WHERE(sid IS NULL OR coverage.source_account_id=sid)AND(coverage.origin='legacy'OR coverage.readiness<>'HISTORICAL_UPDATE_COMPLETE'))),(SELECT count(*)FROM matched)INTO res,total;
 IF page_offset>0 AND page_offset>=total THEN PERFORM banking.collection_error('P2800');END IF;
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation OR character_not_in_repertoire OR untranslatable_character THEN PERFORM banking.collection_error('P2800');END $$;

-- Return only current source detail and independently authorized existing journal anchors.
-- Catalog-only review never projects the whole transaction history for an empty selection.
ALTER FUNCTION banking.collection_rule_context(uuid,uuid,text,boolean)RENAME TO collection_automatic_previous_rule_context;
CREATE FUNCTION banking.collection_rule_context(actor uuid,w uuid,raw text,writing boolean)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;snap jsonb;action_value jsonb;z jsonb;sid uuid;oid uuid;rev bigint;selected jsonb:='[]';row_cache jsonb:='[]';rows_value jsonb:='[]';facts_value jsonb:='[]';fact jsonb;topic_value jsonb;tags_value jsonb;rules_value jsonb;res jsonb;catalog_value jsonb;BEGIN
 IF writing IS NULL THEN PERFORM banking.collection_error('P2800');END IF;
 PERFORM banking.collection_label_authorize(actor,w,writing);v:=banking.import_json(raw,524288);PERFORM banking.collection_keys(v,ARRAY['snapshot','selection','action']);snap:=banking.collection_query_snapshot(w);PERFORM banking.collection_query_pins(v->'snapshot',snap);
 IF NOT writing AND v->'action'<>jsonb_build_object('kind','rules_apply')THEN PERFORM banking.collection_error('P2800');END IF;action_value:=banking.collection_rule_action(w,v->'action');
 IF jsonb_typeof(v->'selection')IS DISTINCT FROM 'array'THEN PERFORM banking.collection_error('P2800');END IF;
 IF jsonb_array_length(v->'selection')>100 OR(writing AND((action_value->>'kind'IN('taxonomy','tag_catalog','rule_catalog')AND jsonb_array_length(v->'selection')<>0)OR(action_value->>'kind'NOT IN('taxonomy','tag_catalog','rule_catalog')AND jsonb_array_length(v->'selection')=0)))THEN PERFORM banking.collection_error('P2805');END IF;
 IF jsonb_array_length(v->'selection')>0 THEN
 SELECT coalesce(jsonb_agg(r.body),'[]')INTO row_cache FROM banking.collection_query_rows(actor,w)r WHERE EXISTS(SELECT FROM jsonb_array_elements(v->'selection')x WHERE(x->>'source_id')::uuid=r.source_id AND(x->>'observation_id')::uuid=r.observation_id);
 END IF;
 FOR z IN SELECT value FROM jsonb_array_elements(v->'selection')LOOP
  PERFORM banking.collection_keys(z,ARRAY['workspace_id','source_id','observation_id','observation_revision']);IF ledger.uuid_value(z->'workspace_id')<>w THEN PERFORM banking.collection_error('P2801');END IF;
  sid:=ledger.uuid_value(z->'source_id');oid:=ledger.uuid_value(z->'observation_id');rev:=banking.import_revision(z->'observation_revision');IF rev=0 THEN PERFORM banking.collection_error('P2800');END IF;
  IF EXISTS(SELECT FROM jsonb_array_elements(selected)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text)THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT x INTO fact FROM jsonb_array_elements(row_cache)x WHERE x->>'source_id'=sid::text AND x->>'observation_id'=oid::text;IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;IF fact->>'observation_revision'<>rev::text THEN PERFORM banking.collection_error('P2803');END IF;
  selected:=selected||jsonb_build_array(jsonb_build_object('workspace_id',w,'source_id',sid,'observation_id',oid,'observation_revision',rev::text));rows_value:=rows_value||jsonb_build_array(fact-ARRAY['institution','legal_owner','rule_application']);
  SELECT jsonb_build_object('observation_revision',d.observation_revision::text,'taxonomy_revision',d.taxonomy_revision::text,'kind',d.kind,'topic_id',d.topic_id)INTO topic_value FROM banking.collection_label_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid ORDER BY decision_revision DESC LIMIT 1;
  SELECT jsonb_build_object('observation_revision',d.observation_revision::text,'tag_revision',d.tag_revision::text,'tag_ids',to_jsonb(d.tag_ids))INTO tags_value FROM banking.collection_tag_decision d WHERE d.workspace_id=w AND d.source_account_id=sid AND d.observation_id=oid ORDER BY decision_revision DESC LIMIT 1;
  facts_value:=facts_value||jsonb_build_array(jsonb_build_object('source_id',sid,'observation_id',oid,'observation_revision',rev::text,'booked_date',fact->'booked_date','merchant',fact->'merchant','description',fact->'description','currency_version_id',fact->'money'->'currency_version_id','direction',fact->'money'->'direction','source_state',fact->'source_state','money_state',fact->'money'->'state','manual_topic',topic_value,'manual_tags',tags_value));
 END LOOP;
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO selected FROM jsonb_array_elements(selected);
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO rows_value FROM jsonb_array_elements(rows_value);
 SELECT coalesce(jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id'),'[]')INTO facts_value FROM jsonb_array_elements(facts_value);
 IF action_value->>'kind'='rule_catalog'THEN rules_value:=action_value->'rules';
  catalog_value:=banking.get_collection_taxonomy_v3(actor,w)||jsonb_build_object('rules',rules_value,'ruleset_revision',((snap->>'ruleset_revision')::bigint+1)::text);
  IF octet_length(catalog_value::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;
 ELSE SELECT e.rules INTO rules_value FROM banking.collection_rule_edition e JOIN banking.collection_label_state s USING(workspace_id,ruleset_revision)WHERE e.workspace_id=w;END IF;
 res:=jsonb_build_object('profile','private-rule-context-v1','principal_id',actor,'workspace_id',w,'snapshot',snap,'action',action_value,'selection',selected,'review_rows',rows_value,'rules',rules_value,'facts',facts_value,
 'topic_ids',(SELECT coalesce(jsonb_agg(t->'topic_id'ORDER BY t->>'topic_id'),'[]')FROM banking.collection_taxonomy_edition e JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(e.topics)t WHERE e.workspace_id=w AND t->>'status'='active'),
 'tag_ids',(SELECT coalesce(jsonb_agg(t->'tag_id'ORDER BY t->>'tag_id'),'[]')FROM banking.collection_tag_edition e JOIN banking.collection_label_state s USING(workspace_id,tag_revision),jsonb_array_elements(e.tags)t WHERE e.workspace_id=w AND t->>'status'='active'));
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;

-- Bounded durable policy, planning and execution entrypoints.

-- Current input inventory excludes pending/removed facts from routine categorization work.
CREATE FUNCTION banking.collection_automatic_inputs(actor uuid,w uuid)
RETURNS TABLE(source_id uuid,observation_id uuid,observation_revision bigint,body jsonb,input_fingerprint text,previous_fingerprint text)
LANGUAGE sql STABLE AS $$
 SELECT r.source_id,r.observation_id,r.observation_revision,r.body,
 banking.collection_automatic_row_fingerprint(w,r.source_id,r.observation_id,r.observation_revision,r.body),d.input_fingerprint
 FROM banking.collection_tag_rows(actor,w)r
 LEFT JOIN LATERAL(SELECT x.input_fingerprint FROM banking.collection_automatic_decision x
  WHERE x.workspace_id=w AND x.source_account_id=r.source_id AND x.observation_id=r.observation_id ORDER BY decision_revision DESC LIMIT 1)d ON true
 WHERE r.body->>'source_state'='booked'
$$;

-- Only currently visible original source aliases may authorize a retained automatic receipt.
CREATE FUNCTION banking.collection_automatic_visible(w uuid,r banking.collection_automatic_run)RETURNS boolean
LANGUAGE sql STABLE SET search_path=pg_catalog,pg_temp AS $$
 SELECT NOT EXISTS(SELECT FROM jsonb_array_elements(r.context->'selection')x
 WHERE NOT EXISTS(SELECT FROM banking.collection_visible_sources(w)s WHERE s.source_account_id=(x->>'source_id')::uuid))
$$;

-- Caller has already acquired principal/book/workspace/source/label locks in the established order.
CREATE FUNCTION banking.collection_automatic_status(actor uuid,w uuid)RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE a banking.collection_automation_state;p banking.collection_automation_edition;r banking.collection_automatic_run;
 pending_value bigint:=0;attention_value bigint:=0;reason_value text;state_value text;last_value jsonb;res jsonb;BEGIN
 SELECT * INTO STRICT a FROM banking.collection_automation_state WHERE workspace_id=w;
 SELECT * INTO STRICT p FROM banking.collection_automation_edition WHERE workspace_id=w AND revision=a.revision;
 reason_value:=a.blocked_reason;
 IF a.last_run_id IS NOT NULL THEN
  SELECT * INTO STRICT r FROM banking.collection_automatic_run WHERE workspace_id=w AND run_id=a.last_run_id;
  IF banking.collection_automatic_visible(w,r)THEN last_value:=banking.collection_automatic_render(r);
  ELSE reason_value:='permission_changed';END IF;
 END IF;
 IF p.enabled THEN
  SELECT count(*)INTO pending_value FROM banking.collection_automatic_inputs(actor,w)i WHERE i.input_fingerprint IS DISTINCT FROM i.previous_fingerprint;
 END IF;
 SELECT count(*)INTO attention_value FROM banking.collection_query_rows(actor,w)q WHERE q.body->>'source_state'='booked'AND q.body->'categorization'->>'state'='needs_attention';
 IF NOT EXISTS(SELECT FROM banking.collection_workspace WHERE workspace_id=w AND 'classify'=ANY(capabilities))THEN reason_value:='permission_changed';END IF;
 IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor)THEN reason_value:='saved_work';END IF;
 state_value:=CASE WHEN NOT p.enabled THEN 'disabled'WHEN reason_value IS NOT NULL THEN 'blocked'
  WHEN pending_value>0 OR EXISTS(SELECT FROM banking.collection_automatic_run WHERE workspace_id=w AND state='prepared')THEN 'pending'ELSE 'idle'END;
 res:=jsonb_build_object('profile','private-collection-automation-v1','workspace_id',w,'revision',a.revision::text,'enabled',p.enabled,
  'processing_state',state_value,'blocked_reason',CASE WHEN p.enabled THEN reason_value ELSE NULL END,
  'pending_count',pending_value::text,'attention_count',attention_value::text,'last_run',last_value);
 IF octet_length(res::text)>16384 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;

CREATE FUNCTION banking.get_collection_automation(actor uuid,w uuid)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$BEGIN
 PERFORM banking.collection_label_authorize(actor,w,false);
 PERFORM 1 FROM banking.collection_automation_state WHERE workspace_id=w FOR SHARE;
 IF NOT FOUND THEN PERFORM banking.collection_error('P2801');END IF;
 RETURN banking.collection_automatic_status(actor,w);
END $$;

CREATE FUNCTION banking.set_collection_automation(actor uuid,w uuid,raw text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE v jsonb;k text;expected bigint;enabled_value boolean;a banking.collection_automation_state;
 old banking.collection_automation_setting_receipt;prior_run banking.collection_automatic_run;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);
 SELECT * INTO STRICT a FROM banking.collection_automation_state WHERE workspace_id=w FOR UPDATE;
 v:=banking.import_json(raw,1024);PERFORM banking.collection_keys(v,ARRAY['key','expected_revision','enabled']);
 k:=ledger.text_value(v->'key');expected:=banking.import_revision(v->'expected_revision');
 IF k COLLATE "C" !~ '^[A-Za-z0-9._:-]{1,128}$'OR jsonb_typeof(v->'enabled')IS DISTINCT FROM 'boolean'THEN PERFORM banking.collection_error('P2800');END IF;
 enabled_value:=(v->>'enabled')::boolean;
 SELECT * INTO old FROM banking.collection_automation_setting_receipt WHERE workspace_id=w AND principal_id=actor AND request_key=k;
 IF FOUND THEN
  IF old.request<>v THEN PERFORM banking.collection_error('P2802');END IF;
  IF old.result->'last_run'<>'null'::jsonb THEN
   SELECT * INTO STRICT prior_run FROM banking.collection_automatic_run WHERE workspace_id=w AND run_id=(old.result->'last_run'->>'run_id')::uuid;
   IF NOT banking.collection_automatic_visible(w,prior_run)THEN PERFORM banking.collection_error('P2801');END IF;
  END IF;
  RETURN old.result;
 END IF;
 IF a.revision<>expected THEN PERFORM banking.collection_error('P2803');END IF;
 INSERT INTO banking.collection_automation_edition(workspace_id,revision,enabled,principal_id)VALUES(w,a.revision+1,enabled_value,actor);
 UPDATE banking.collection_automation_state SET revision=revision+1,blocked_reason=NULL WHERE workspace_id=w;
 UPDATE banking.collection_label_state SET projection_revision=projection_revision+1 WHERE workspace_id=w;
 res:=banking.collection_automatic_status(actor,w);
 INSERT INTO banking.collection_automation_setting_receipt(workspace_id,principal_id,request_key,request,result)VALUES(w,actor,k,v,res);
 RETURN res;
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;

-- Work metadata is available only to the restricted API service, never a public request selector.
-- Include disabled-policy prepared runs so disabling cannot strand an inert plan forever.
CREATE FUNCTION banking.collection_automation_work()RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE x record;items jsonb:='[]';BEGIN
 IF NOT integration.provider_api_login(session_user)THEN PERFORM banking.collection_error('P2801');END IF;
 IF current_setting('transaction_isolation')<>'serializable'THEN PERFORM banking.collection_error('P2800');END IF;
 FOR x IN SELECT chosen.* FROM(
  SELECT c.owner_principal_id actor,c.workspace_id w,a.last_checked_at
  FROM banking.collection_workspace c JOIN banking.collection_automation_state a USING(workspace_id)
  JOIN banking.collection_automation_edition p ON p.workspace_id=a.workspace_id AND p.revision=a.revision
  JOIN ledger.principal principal ON principal.principal_id=c.owner_principal_id AND principal.disabled_at IS NULL
  WHERE 'read'=ANY(c.capabilities)AND 'classify'=ANY(c.capabilities)
   AND(p.enabled OR EXISTS(SELECT FROM banking.collection_automatic_run r WHERE r.workspace_id=c.workspace_id AND r.state='prepared'))
  ORDER BY a.last_checked_at NULLS FIRST,c.workspace_id LIMIT 10
 )chosen ORDER BY chosen.actor,chosen.w LOOP
  BEGIN
   PERFORM banking.collection_label_authorize(x.actor,x.w,true);
   PERFORM 1 FROM banking.collection_automation_state WHERE workspace_id=x.w FOR UPDATE;
   -- Inventory rotation commits independently of a later failed/unknown workspace phase.
   UPDATE banking.collection_automation_state SET last_checked_at=clock_timestamp()WHERE workspace_id=x.w;
   items:=items||jsonb_build_array(jsonb_build_object('principal_id',x.actor,'workspace_id',x.w));
  EXCEPTION WHEN SQLSTATE 'P2801'OR SQLSTATE 'P2804' THEN CONTINUE;END;
 END LOOP;
 RETURN jsonb_build_object('profile','private-collection-auto-work-v1','workspaces',items);
END $$;

CREATE FUNCTION banking.collection_automation_context(actor uuid,w uuid)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a banking.collection_automation_state;p banking.collection_automation_edition;r banking.collection_automatic_run;
 selected jsonb;context_value jsonb;last_value jsonb;state_value text:='idle';res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);
 SELECT * INTO STRICT a FROM banking.collection_automation_state WHERE workspace_id=w FOR UPDATE;
 SELECT * INTO STRICT p FROM banking.collection_automation_edition WHERE workspace_id=w AND revision=a.revision;
 IF a.last_run_id IS NOT NULL THEN
  SELECT * INTO STRICT r FROM banking.collection_automatic_run WHERE workspace_id=w AND run_id=a.last_run_id;
  IF NOT banking.collection_automatic_visible(w,r)THEN
   UPDATE banking.collection_automation_state SET blocked_reason='permission_changed',last_checked_at=clock_timestamp()WHERE workspace_id=w;
   RETURN jsonb_build_object('profile','private-collection-auto-context-v1','principal_id',actor,'workspace_id',w,'state','blocked','run_id',NULL,'context',NULL,'last_run',NULL);
  END IF;
  last_value:=banking.collection_automatic_render(r);
 END IF;
 IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor)THEN
  UPDATE banking.collection_automation_state SET blocked_reason='saved_work',last_checked_at=clock_timestamp()WHERE workspace_id=w;
  state_value:='blocked';
 ELSE
  UPDATE banking.collection_automation_state SET blocked_reason=NULL,last_checked_at=clock_timestamp()WHERE workspace_id=w;
  SELECT * INTO r FROM banking.collection_automatic_run WHERE workspace_id=w AND state='prepared'FOR UPDATE;
  IF FOUND THEN
   IF NOT banking.collection_automatic_visible(w,r)THEN PERFORM banking.collection_error('P2801');END IF;
   RETURN jsonb_build_object('profile','private-collection-auto-context-v1','principal_id',actor,'workspace_id',w,'state','pending','run_id',r.run_id,'context',NULL,'last_run',banking.collection_automatic_render(r));
  ELSIF p.enabled THEN
   IF NOT banking.collection_automatic_starter(actor,w)THEN
    UPDATE banking.collection_automation_state SET blocked_reason='category_catalog_unavailable'WHERE workspace_id=w;state_value:='blocked';
   ELSE
    SELECT coalesce(jsonb_agg(jsonb_build_object('workspace_id',w,'source_id',i.source_id,'observation_id',i.observation_id,'observation_revision',i.observation_revision::text)ORDER BY i.source_id,i.observation_id),'[]')INTO selected
    FROM(SELECT * FROM banking.collection_automatic_inputs(actor,w)x WHERE x.input_fingerprint IS DISTINCT FROM x.previous_fingerprint ORDER BY x.source_id,x.observation_id LIMIT 100)i;
    IF jsonb_array_length(selected)>0 THEN
     context_value:=banking.collection_rule_context(actor,w,jsonb_build_object('snapshot',banking.collection_query_snapshot(w),'selection',selected,'action',jsonb_build_object('kind','rules_apply'))::text,true);
     state_value:='ready';
    END IF;
   END IF;
  END IF;
 END IF;
 res:=jsonb_build_object('profile','private-collection-auto-context-v1','principal_id',actor,'workspace_id',w,'state',state_value,'run_id',NULL,'context',context_value,'last_run',last_value);
 IF octet_length(res::text)>1048576 THEN PERFORM banking.collection_error('P2805');END IF;RETURN res;
END $$;

CREATE FUNCTION banking.prepare_collection_automatic_run(actor uuid,w uuid,context_text text,results_text text)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a banking.collection_automation_state;p banking.collection_automation_edition;r banking.collection_automatic_run;
 c jsonb;fresh jsonb;v jsonb;x jsonb;input_cache jsonb;normalized jsonb:='[]';fingerprints jsonb:='[]';key_value text;sid uuid;oid uuid;fp text;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);
 SELECT * INTO STRICT a FROM banking.collection_automation_state WHERE workspace_id=w FOR UPDATE;
 SELECT * INTO STRICT p FROM banking.collection_automation_edition WHERE workspace_id=w AND revision=a.revision;
 IF NOT p.enabled OR NOT a.initialized OR EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor)THEN PERFORM banking.collection_error('P2804');END IF;
 c:=banking.import_json(context_text,1048576);v:=banking.import_json(results_text,1048576);
 PERFORM banking.collection_keys(v,ARRAY['results']);
 IF c->>'profile'<>'private-rule-context-v1'OR c->>'principal_id'<>actor::text OR c->>'workspace_id'<>w::text OR c->'action'<>jsonb_build_object('kind','rules_apply')THEN PERFORM banking.collection_error('P2800');END IF;
 fresh:=banking.collection_rule_context(actor,w,jsonb_build_object('snapshot',c->'snapshot','selection',c->'selection','action',c->'action')::text,true);
 IF fresh<>c THEN PERFORM banking.collection_error('P2803');END IF;
 IF jsonb_typeof(v->'results')IS DISTINCT FROM 'array'OR jsonb_array_length(v->'results')<>jsonb_array_length(c->'facts')OR jsonb_array_length(v->'results')NOT BETWEEN 1 AND 100 THEN PERFORM banking.collection_error('P2800');END IF;
 SELECT coalesce(jsonb_agg(jsonb_build_object('source_id',i.source_id,'observation_id',i.observation_id,'fingerprint',i.input_fingerprint)),'[]')INTO input_cache FROM banking.collection_automatic_inputs(actor,w)i WHERE EXISTS(SELECT FROM jsonb_array_elements(c->'selection')z WHERE z->>'source_id'=i.source_id::text AND z->>'observation_id'=i.observation_id::text);
 FOR x IN SELECT value FROM jsonb_array_elements(v->'results')LOOP
  x:=banking.collection_rule_result(w,c,x);sid:=(x->>'source_id')::uuid;oid:=(x->>'observation_id')::uuid;
  IF EXISTS(SELECT FROM jsonb_array_elements(normalized)n WHERE n->>'source_id'=sid::text AND n->>'observation_id'=oid::text)THEN PERFORM banking.collection_error('P2800');END IF;
  SELECT z->>'fingerprint'INTO fp FROM jsonb_array_elements(input_cache)z WHERE z->>'source_id'=sid::text AND z->>'observation_id'=oid::text;IF fp IS NULL THEN PERFORM banking.collection_error('P2801');END IF;
  normalized:=normalized||jsonb_build_array(x);
  fingerprints:=fingerprints||jsonb_build_array(jsonb_build_object('source_id',sid,'observation_id',oid,'fingerprint',fp));
 END LOOP;
 SELECT jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id')INTO normalized FROM jsonb_array_elements(normalized);
 SELECT jsonb_agg(value ORDER BY value->>'source_id',value->>'observation_id')INTO fingerprints FROM jsonb_array_elements(fingerprints);
 key_value:=encode(sha256(convert_to(jsonb_build_array(actor,w,a.revision::text,fingerprints,normalized)::text,'UTF8')),'hex');
 SELECT * INTO r FROM banking.collection_automatic_run WHERE workspace_id=w AND work_key=key_value;
 IF FOUND THEN RETURN banking.collection_automatic_render(r);END IF;
 IF EXISTS(SELECT FROM banking.collection_automatic_run WHERE workspace_id=w AND state='prepared')THEN PERFORM banking.collection_error('P2802');END IF;
 INSERT INTO banking.collection_automatic_run(workspace_id,work_key,principal_id,policy_revision,context,evaluations,input_fingerprints)
 VALUES(w,key_value,actor,a.revision,c,normalized,fingerprints)RETURNING * INTO r;
 UPDATE banking.collection_automation_state SET last_run_id=r.run_id,blocked_reason=NULL WHERE workspace_id=w;
 RETURN banking.collection_automatic_render(r);
EXCEPTION WHEN SQLSTATE 'P2100'OR SQLSTATE 'P2400'OR invalid_text_representation THEN PERFORM banking.collection_error('P2800');END $$;

-- Decisions contain only analytic output/provenance; this helper never invokes ledger posting.
CREATE FUNCTION banking.collection_automatic_classification(w uuid,c jsonb,evaluation jsonb)RETURNS jsonb
LANGUAGE plpgsql SET search_path=pg_catalog,pg_temp AS $$
DECLARE sid uuid:=(evaluation->>'source_id')::uuid;oid uuid:=(evaluation->>'observation_id')::uuid;rev bigint:=(evaluation->>'observation_revision')::bigint;
 fact jsonb;metadata jsonb;provider_value jsonb;category_value jsonb;topic_value uuid;output_value jsonb;reason_value text;
 origin_value text:='none';state_value text:='needs_attention';names_value jsonb:='[]';status_value text:=evaluation->>'status';BEGIN
 SELECT f INTO STRICT fact FROM jsonb_array_elements(c->'facts')f WHERE f->>'source_id'=sid::text AND f->>'observation_id'=oid::text;
 SELECT coalesce(jsonb_agg(rule_row->'name'ORDER BY rule_row->>'rule_id'),'[]')INTO names_value FROM jsonb_array_elements(c->'rules')rule_row
 WHERE EXISTS(SELECT FROM jsonb_array_elements(evaluation->'rules')winner WHERE winner->>'rule_id'=rule_row->>'rule_id');
 IF status_value='stale_override'THEN reason_value:='stale_override';
 ELSIF status_value='override'THEN state_value:='manual';origin_value:='manual';output_value:=evaluation->'output';
 ELSIF status_value='rule'THEN state_value:='automatic';origin_value:='rule';output_value:=evaluation->'output';
 ELSIF status_value='conflict'THEN reason_value:='conflicting_rules';
 ELSIF status_value='unavailable'THEN state_value:='unavailable';reason_value:=CASE WHEN fact->>'source_state'<>'booked'THEN 'source_not_ready'ELSE 'unavailable'END;
 ELSIF fact->'manual_topic'<>'null'::jsonb THEN
  state_value:='manual';origin_value:='manual';output_value:=jsonb_build_object('kind',fact->'manual_topic'->>'kind','tag_ids',coalesce(fact->'manual_tags'->'tag_ids','[]'));
  IF fact->'manual_topic'->>'kind'='topic'THEN output_value:=output_value||jsonb_build_object('topic_id',fact->'manual_topic'->>'topic_id');END IF;
 ELSE
  metadata:=banking.collection_automatic_provider(w,sid,oid,rev);category_value:=metadata->'category';
  IF metadata IS NULL THEN reason_value:='missing_provider_category';
  ELSE
   provider_value:=jsonb_build_object('name','Plaid','version',category_value->'version','primary',category_value->'primary','detailed',category_value->'detailed','confidence',category_value->'confidence','map_edition','plaid-pfc-primary-2026-11-11');
   IF EXISTS(SELECT FROM unnest(ARRAY['version','primary','detailed','confidence'])k WHERE category_value->k='null'::jsonb)THEN provider_value:=NULL;END IF;
   reason_value:=nullif(category_value->>'unavailable_reason','');
   IF reason_value IN('not_provided','null_category','incomplete_category')THEN reason_value:='missing_provider_category';END IF;
   IF reason_value IS NULL AND category_value->>'confidence'NOT IN('HIGH','VERY_HIGH')THEN reason_value:='low_confidence';END IF;
   IF reason_value IS NULL THEN
    SELECT a.topic_id INTO topic_value FROM reference.analytic_category_pair pair JOIN reference.analytic_category a ON a.code=pair.primary_code
    WHERE pair.version=category_value->>'version'AND pair.primary_code=category_value->>'primary'AND pair.detailed_code=category_value->>'detailed'AND a.map_edition='plaid-pfc-primary-2026-11-11';
    IF topic_value IS NULL THEN reason_value:='unmapped_category';
    ELSIF NOT EXISTS(SELECT FROM banking.collection_taxonomy_edition t JOIN banking.collection_label_state s USING(workspace_id,taxonomy_revision),jsonb_array_elements(t.topics)x WHERE t.workspace_id=w AND x->>'topic_id'=topic_value::text AND x->>'status'='active')THEN reason_value:='retired_category';
    ELSE state_value:='automatic';origin_value:='provider';output_value:=jsonb_build_object('kind','topic','topic_id',topic_value,'tag_ids',coalesce(fact->'manual_tags'->'tag_ids','[]'));END IF;
   END IF;
  END IF;
 END IF;
 RETURN jsonb_build_object('output',output_value,'classification',jsonb_build_object('state',state_value,'origin',origin_value,'reason',reason_value,'rule_names',names_value,'provider',provider_value));
END $$;

CREATE FUNCTION banking.execute_collection_automatic_run(actor uuid,w uuid,id uuid)RETURNS jsonb
LANGUAGE plpgsql SECURITY DEFINER SET search_path=pg_catalog,pg_temp AS $$
DECLARE a banking.collection_automation_state;p banking.collection_automation_edition;r banking.collection_automatic_run;
 x jsonb;input_cache jsonb;fp text;classified jsonb;next_revision bigint;attention_value bigint:=0;processed_value bigint:=0;stale boolean:=false;res jsonb;BEGIN
 PERFORM banking.collection_label_authorize(actor,w,true);
 SELECT * INTO STRICT a FROM banking.collection_automation_state WHERE workspace_id=w FOR UPDATE;
 SELECT * INTO STRICT p FROM banking.collection_automation_edition WHERE workspace_id=w AND revision=a.revision;
 SELECT * INTO r FROM banking.collection_automatic_run WHERE workspace_id=w AND run_id=id AND principal_id=actor FOR UPDATE;
 IF NOT FOUND OR NOT banking.collection_automatic_visible(w,r)THEN PERFORM banking.collection_error('P2801');END IF;
 IF r.state<>'prepared'THEN RETURN banking.collection_automatic_render(r);END IF;
 IF EXISTS(SELECT FROM ledger.interactive_command_reservation WHERE principal_id=actor)THEN PERFORM banking.collection_error('P2804');END IF;
 stale:=NOT p.enabled OR a.revision<>r.policy_revision;
 SELECT coalesce(jsonb_agg(jsonb_build_object('source_id',i.source_id,'observation_id',i.observation_id,'fingerprint',i.input_fingerprint)),'[]')INTO input_cache FROM banking.collection_automatic_inputs(actor,w)i WHERE EXISTS(SELECT FROM jsonb_array_elements(r.input_fingerprints)z WHERE z->>'source_id'=i.source_id::text AND z->>'observation_id'=i.observation_id::text);
 FOR x IN SELECT value FROM jsonb_array_elements(r.input_fingerprints)LOOP
  SELECT z->>'fingerprint'INTO fp FROM jsonb_array_elements(input_cache)z WHERE z->>'source_id'=x->>'source_id'AND z->>'observation_id'=x->>'observation_id';
  IF fp IS DISTINCT FROM x->>'fingerprint'THEN stale:=true;END IF;
 END LOOP;
 IF NOT stale THEN
  SELECT decision_revision+1 INTO STRICT next_revision FROM banking.collection_label_state WHERE workspace_id=w;
  FOR x IN SELECT value FROM jsonb_array_elements(r.evaluations)LOOP
   classified:=banking.collection_automatic_classification(w,r.context,x);
   SELECT z->>'fingerprint'INTO STRICT fp FROM jsonb_array_elements(r.input_fingerprints)z WHERE z->>'source_id'=x->>'source_id'AND z->>'observation_id'=x->>'observation_id';
   INSERT INTO banking.collection_automatic_decision(workspace_id,source_account_id,observation_id,observation_revision,decision_revision,run_id,input_fingerprint,output,classification,rules,source_attribution,policy_revision)
   VALUES(w,(x->>'source_id')::uuid,(x->>'observation_id')::uuid,(x->>'observation_revision')::bigint,next_revision,id,fp,nullif(classified->'output','null'::jsonb),classified->'classification',x->'rules',banking.collection_automatic_provider(w,(x->>'source_id')::uuid,(x->>'observation_id')::uuid,(x->>'observation_revision')::bigint),r.policy_revision);
   processed_value:=processed_value+1;
  END LOOP;
  -- The existing immutable state guard accepts these separate monotonic advances.
  UPDATE banking.collection_label_state SET decision_revision=decision_revision+1 WHERE workspace_id=w;
  UPDATE banking.collection_label_state SET projection_revision=projection_revision+1 WHERE workspace_id=w;
  -- Owner-priority projection, not raw evaluator taxonomy staleness, owns visible attention.
  SELECT count(*)INTO attention_value FROM banking.collection_query_rows(actor,w)q
  WHERE q.body->>'source_state'='booked'AND q.body->'categorization'->>'state'='needs_attention'
   AND EXISTS(SELECT FROM jsonb_array_elements(r.evaluations)e WHERE e->>'source_id'=q.source_id::text AND e->>'observation_id'=q.observation_id::text);
 END IF;
 res:=jsonb_build_object('profile','private-collection-auto-run-v1','workspace_id',w,'run_id',id,'state',CASE WHEN stale THEN 'stale'ELSE 'committed'END,
  'processed_count',processed_value::text,'attention_count',attention_value::text);
 UPDATE banking.collection_automatic_run SET state=CASE WHEN stale THEN 'stale'ELSE 'committed'END,result=res,terminal_at=clock_timestamp()WHERE workspace_id=w AND run_id=id RETURNING * INTO r;
 UPDATE banking.collection_automation_state SET last_run_id=id,blocked_reason=NULL,last_checked_at=clock_timestamp()WHERE workspace_id=w;
 RETURN banking.collection_automatic_render(r);
END $$;

-- AUTOMATION-PRIVILEGES
DO $$DECLARE f regprocedure;BEGIN
 FOR f IN SELECT p.oid::regprocedure FROM pg_proc p JOIN pg_namespace n ON n.oid=p.pronamespace
 WHERE (n.nspname='banking'AND(p.proname LIKE 'collection_automatic_%'OR p.proname LIKE 'collection_automation_%'
 OR p.proname IN('automatic_category_metadata','get_collection_automation','set_collection_automation','prepare_collection_automatic_run','execute_collection_automatic_run')))
 OR(n.nspname='integration'AND p.proname IN('stage_sync_category_page','sync_category_context','complete_sync_categories'))
 LOOP EXECUTE format('REVOKE ALL ON FUNCTION %s FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator',f);END LOOP;
END $$;
REVOKE ALL ON FUNCTION banking.collection_rule_context(uuid,uuid,text,boolean),banking.query_collection_review_v3(uuid,uuid,text)FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION banking.collection_rule_context(uuid,uuid,text,boolean),banking.query_collection_review_v3(uuid,uuid,text)TO bookkeeping_api;
REVOKE ALL ON reference.analytic_category,reference.analytic_category_pair,integration.sync_category_page,integration.sync_category_completion,banking.provider_category_revision,banking.collection_automation_edition,banking.collection_automation_state,banking.collection_automation_setting_receipt,banking.collection_automatic_run,banking.collection_automatic_decision FROM PUBLIC,bookkeeping_api,bookkeeping_sync,bookkeeping_webhook,bookkeeping_observer,bookkeeping_publisher,bookkeeping_verifier,bookkeeping_identity_operator;
GRANT EXECUTE ON FUNCTION integration.stage_sync_category_page(uuid,uuid,uuid,uuid,bigint,text),integration.sync_category_context(uuid,uuid,uuid,uuid,bigint),integration.complete_sync_categories(uuid,uuid,uuid,uuid,bigint)TO bookkeeping_sync;
GRANT EXECUTE ON FUNCTION banking.get_collection_automation(uuid,uuid),banking.set_collection_automation(uuid,uuid,text),banking.collection_automation_work(),banking.collection_automation_context(uuid,uuid),banking.prepare_collection_automatic_run(uuid,uuid,text,text),banking.execute_collection_automatic_run(uuid,uuid,uuid)TO bookkeeping_api;
COMMIT;
