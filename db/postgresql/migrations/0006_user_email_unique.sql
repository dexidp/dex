-- +migrate Up
ALTER TABLE ONLY authd_user
    ADD CONSTRAINT authd_user_email_key UNIQUE (email);
