-- +migrate Up
ALTER TABLE key ADD COLUMN tmp_value bytea;
UPDATE KEY SET tmp_value = value;
ALTER TABLE key DROP COLUMN value;
ALTER TABLE key RENAME COLUMN "tmp_value" to "value";
