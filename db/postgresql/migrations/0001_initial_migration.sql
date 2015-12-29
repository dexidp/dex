-- +migrate Up
CREATE TABLE IF NOT EXISTS "authd_user" (
       "id" text not null primary key,
       "email" text,
       "email_verified" boolean,
       "display_name" text,
       "admin" boolean) ;

CREATE TABLE IF NOT EXISTS "client_identity" (
       "id" text not null primary key,
       "secret" bytea,
       "metadata" text);

CREATE TABLE IF NOT EXISTS "connector_config" (
       "id" text not null primary key,
       "type" text, "config" text) ;

CREATE TABLE IF NOT EXISTS "key" (
       "value" bytea not null primary key) ;

CREATE TABLE IF NOT EXISTS "password_info" (
       "user_id" text not null primary key,
       "password" text,
       "password_expires" bigint) ;

CREATE TABLE IF NOT EXISTS "session" (
       "id" text not null primary key,
       "state" text,
       "created_at" bigint,
       "expires_at" bigint,
       "client_id" text,
       "client_state" text,
       "redirect_url" text, "identity" text,
       "connector_id" text,
       "user_id" text, "register" boolean) ;

CREATE TABLE IF NOT EXISTS "session_key" (
       "key" text not null primary key,
       "session_id" text,
       "expires_at" bigint,
       "stale" boolean) ;

CREATE TABLE IF NOT EXISTS "remote_identity_mapping" (
       "connector_id" text not null,
       "user_id" text,
       "remote_id" text not null,
       primary key ("connector_id", "remote_id")) ;
