-- +migrate Up
ALTER TABLE session ADD COLUMN "scope" text;
