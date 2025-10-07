CREATE SEQUENCE transaction_id_seq;

CREATE TABLE public.transaction (
    id integer DEFAULT NEXTVAL('public.transaction_id_seq' :: regclass) NOT NULL CONSTRAINT transaction_pk PRIMARY KEY,
    "transactionDate" date NOT NULL,
    "fromAccount" text NOT NULL,
    "fromNarrative" text NOT NULL,
    "toAccount" text NOT NULL,
    "toNarrative" text NOT NULL,
    amount numeric(23, 8),
    "status" text NOT NULL,
    "method" text NOT NULL,
    "typeTransaction" text NOT NULL,
    "description" text,
    "refNumber" varchar(100),
    "transactionId" TEXT NOT NULL UNIQUE,
    "orderTime" TIMESTAMP WITH TIME ZONE NULL,
    "orderType" VARCHAR(50) NULL,
    "transactionTime" TIMESTAMP WITH TIME ZONE NULL,
    "currency" VARCHAR(10) NULL,
    "metadata" JSONB NULL DEFAULT '{}' :: JSONB,
    "createdAt" timestamp WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "updatedAt" timestamp WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX transaction_date_index ON transaction ("transactionDate");

CREATE INDEX transaction_ref_number_index ON transaction ("refNumber");

CREATE SEQUENCE account_id_seq;

CREATE TABLE public.account (
    id integer DEFAULT NEXTVAL('public.account_id_seq' :: regclass) NOT NULL CONSTRAINT account_pk PRIMARY KEY,
    "accountNumber" text UNIQUE NOT NULL,
    "name" varchar(100),
    "ownerId" VARCHAR(15),
    "altId" varchar(50),
    "legacyId" JSONB NULL,
    "actualBalance" numeric(23, 8) NOT NULL DEFAULT 0,
    "pendingBalance" numeric(23, 8) NOT NULL DEFAULT 0,
    "categoryCode" varchar(5),
    "subCategoryCode" varchar(5),
    "entityCode" varchar(5),
    "currency" varchar(3),
    "isHvt" boolean NOT NULL DEFAULT false,
    "status" varchar(15),
    "version" INT NOT NULL DEFAULT 1,
    "createdAt" timestamp WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "updatedAt" timestamp WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE INDEX account_account_number_index ON account ("accountNumber");

CREATE INDEX account_version_index ON account ("version");

CREATE TABLE public.account_balance_daily (
    "accountNumber" text NOT NULL,
    "date" date NOT NULL,
    "balance" numeric(23, 8) NOT NULL,
    "createdAt" timestamp WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "updatedAt" timestamp WITH TIME ZONE DEFAULT NOW() NOT NULL,
    CONSTRAINT account_balance_daily_unique_constraint UNIQUE ("accountNumber", "date")
);

CREATE INDEX account_balance_daily_accountNumber_date_idx ON public.account_balance_daily USING btree ("accountNumber", "date");

CREATE TABLE public.category (
    id SERIAL PRIMARY KEY,
    code VARCHAR(5) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(50),
    "createdAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    "updatedAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE public.sub_category (
    id SERIAL PRIMARY KEY,
    code VARCHAR(5) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(50),
    "createdAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    "updatedAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    "categoryCode" VARCHAR(5) NOT NULL,
    FOREIGN KEY ("categoryCode") REFERENCES public.category(code)
);

CREATE TABLE public.entity (
    id SERIAL PRIMARY KEY,
    code VARCHAR(5) UNIQUE NOT NULL,
    name VARCHAR(50) NOT NULL,
    description VARCHAR(50),
    "createdAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    "updatedAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE TABLE IF NOT EXISTS public.recon_tool_history (
    id serial PRIMARY KEY,
    "orderType" varchar(50),
    "transactionType" varchar(50),
    "transactionDate" date,
    "resultFilePath" varchar(255),
    "uploadedFilePath" varchar(255),
    status varchar(50),
    "createdAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    "updatedAt" TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE public.wallet_transaction (
    id UUID DEFAULT uuid_generate_v4() UNIQUE,
    "accountNumber" VARCHAR(50) NOT NULL,
    "refNumber" VARCHAR(255) NOT NULL,
    "transactionType" VARCHAR(50) NOT NULL,
    "transactionFlow" VARCHAR(50) NOT NULL,
    "transactionTime" TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "netAmount" NUMERIC(23, 8) NOT NULL,
    "breakdownAmounts" JSONB NOT NULL,
    "status" VARCHAR(15) NOT NULL,
    "destinationAccountNumber" VARCHAR(50) NULL,
    "description" VARCHAR(255) NULL,
    "metadata" JSONB NULL DEFAULT '{}' :: JSONB,
    "createdAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    "updatedAt" TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

CREATE TABLE IF NOT EXISTS public.feature (
    account_number VARCHAR(64),
    preset TEXT,
    balance_range_min NUMERIC(15, 2) DEFAULT 0,
    negative_balance_allowed BOOLEAN DEFAULT FALSE,
    negative_balance_limit NUMERIC(15, 2) DEFAULT 0,
    created_on TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    updated_on TIMESTAMPTZ,
    PRIMARY KEY (account_number)
);


--
CREATE INDEX IF NOT EXISTS wallet_transaction_account_number_index ON wallet_transaction("accountNumber");

CREATE INDEX CONCURRENTLY IF NOT EXISTS wallet_transaction_for_filtering_data_index ON wallet_transaction("accountNumber", "transactionType", "transactionTime", "id", "refNumber");
CREATE INDEX CONCURRENTLY IF NOT EXISTS wallet_transaction_destination_account_number_for_filtering_data_index ON wallet_transaction("destinationAccountNumber", "transactionType", "transactionTime", "id", "refNumber");
CREATE INDEX CONCURRENTLY IF NOT EXISTS wallet_transaction_refNumber_index ON wallet_transaction("refNumber");
CREATE INDEX CONCURRENTLY IF NOT EXISTS transaction_composite_filter ON transaction("orderType", "typeTransaction", "transactionDate", "refNumber", "id");

ALTER TABLE public.account
    ADD COLUMN IF NOT EXISTS "metadata" JSONB NULL DEFAULT '{}'::JSONB,
    ADD COLUMN IF NOT EXISTS "productTypeName" VARCHAR(255) NULL DEFAULT '';

-- create INDEX account.name relate task ATRX-1014
CREATE INDEX IF NOT EXISTS idx_account_name_lower ON account (LOWER(name));
