-- +migrate Up
ALTER TABLE session ADD COLUMN "nonce" text;
