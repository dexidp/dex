-- +migrate Up
ALTER TABLE authd_user ADD COLUMN disabled boolean;

UPDATE authd_user SET "disabled" = FALSE;
