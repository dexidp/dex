-- +migrate Up
ALTER TABLE refresh_token ADD COLUMN "scopes" text;

UPDATE refresh_token SET scopes = 'openid profile email offline_access';
