-- +migrate Up
ALTER TABLE authd_user ADD COLUMN "created_at" bigint;

UPDATE authd_user SET "created_at" = 0;
