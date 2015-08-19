-- +migrate Up
ALTER TABLE authd_user ADD COLUMN "created_at" bigint;
