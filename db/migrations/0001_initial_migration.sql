-- +migrate Up
CREATE TABLE IF NOT EXISTS "authd_user" (
    "id" text not null primary key,
    "email" text UNIQUE,
    "email_verified" boolean,
    "first_name" text,
    "last_name" text,
    "display_name" text,
    "organization_id" text,
    "admin" boolean,
    "created_at" bigint,
    "disabled" boolean
);

CREATE TABLE IF NOT EXISTS "client_identity" (
    "id" text not null primary key,
    "secret" bytea,
    "metadata" text,
    "dex_admin" boolean,
    "public" boolean
);

CREATE TABLE IF NOT EXISTS "connector_config" (
    "id" text not null primary key,
    "type" text,
    "config" text
);

CREATE TABLE IF NOT EXISTS "key" (
    "value" bytea not null
);

CREATE TABLE IF NOT EXISTS "password_info" (
    "user_id" text not null primary key,
    "password" text,
    "password_expires" bigint
);

CREATE TABLE IF NOT EXISTS "session" (
    "id" text not null primary key,
    "state" text,
    "created_at" bigint,
    "expires_at" bigint,
    "client_id" text,
    "client_state" text,
    "redirect_url" text, 
    "identity" text,
    "connector_id" text,
    "user_id" text, 
    "register" boolean,
    "nonce" text,
    "scope" text,
    "groups" text
);

CREATE TABLE IF NOT EXISTS "session_key" (
    "key" text not null primary key,
    "session_id" text,
    "expires_at" bigint,
    "stale" boolean
);

CREATE TABLE IF NOT EXISTS "remote_identity_mapping" (
    "connector_id" text not null,
    "user_id" text,
    "remote_id" text not null,
    primary key ("connector_id", "remote_id")
);

CREATE TABLE IF NOT EXISTS "refresh_token" (
    "id" bigint NOT NULL,
    "payload_hash" bytea,
    "user_id" text,
    "client_id" text,
    "scopes" text,
    "connector_id" text
);

CREATE SEQUENCE refresh_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE refresh_token_id_seq OWNED BY refresh_token.id;

ALTER TABLE ONLY refresh_token ALTER COLUMN id SET DEFAULT nextval('refresh_token_id_seq'::regclass);

ALTER TABLE ONLY refresh_token ADD CONSTRAINT refresh_token_pkey PRIMARY KEY (id);

CREATE TABLE IF NOT EXISTS "trusted_peers" (
    "client_id" text not null,
    "trusted_client_id" text not null,
    primary key ("client_id", "trusted_client_id")
);

CREATE TABLE IF NOT EXISTS "organization" (
    "organization_id" text not null primary key,
    "name" text UNIQUE,
    "owner_id" text,
    "created_at" bigint,
    "disabled" boolean
);

CREATE INDEX authd_user_organization_index ON authd_user (organization_id);
