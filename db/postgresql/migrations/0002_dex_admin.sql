-- +migrate Up
ALTER TABLE client_identity ADD COLUMN "dex_admin" boolean;

UPDATE "client_identity" SET "dex_admin" = false;
