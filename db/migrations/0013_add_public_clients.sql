-- +migrate Up
ALTER TABLE client_identity ADD COLUMN "public" boolean;

UPDATE "client_identity" SET "public" = false;
