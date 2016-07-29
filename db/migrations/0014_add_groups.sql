-- +migrate Up
ALTER TABLE refresh_token ADD COLUMN "connector_id" text;
ALTER TABLE session ADD COLUMN "groups" text;
