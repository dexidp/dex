-- +migrate Up
ALTER TABLE client_identity ADD COLUMN "dex_admin" boolean;
