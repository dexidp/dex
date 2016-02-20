package db

// SQLite3 is a test only database. There is only one migration because we do not support migrations.
const sqlite3Migration = `
CREATE TABLE authd_user (
    id text NOT NULL UNIQUE,
    email text,
    email_verified integer,
    display_name text,
    admin integer,
    created_at bigint,
    disabled integer
);

CREATE TABLE client_identity (
    id text NOT NULL UNIQUE,
    secret blob,
    metadata text,
    dex_admin integer
);

CREATE TABLE connector_config (
    id text NOT NULL UNIQUE,
    type text,
    config text
);

CREATE TABLE key (
    value blob
);

CREATE TABLE password_info (
    user_id text NOT NULL UNIQUE,
    password text,
    password_expires bigint
);

CREATE TABLE refresh_token (
    id integer PRIMARY KEY,
    payload_hash blob,
    user_id text,
    client_id text
);

CREATE TABLE remote_identity_mapping (
    connector_id text NOT NULL,
    user_id text,
    remote_id text NOT NULL
);

CREATE TABLE session (
    id text NOT NULL UNIQUE,
    state text,
    created_at bigint,
    expires_at bigint,
    client_id text,
    client_state text,
    redirect_url text,
    identity text,
    connector_id text,
    user_id text,
    register integer,
    nonce text,
    scope text
);

CREATE TABLE session_key (
    key text NOT NULL,
    session_id text,
    expires_at bigint,
    stale integer
);
`
