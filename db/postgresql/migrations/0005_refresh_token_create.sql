-- +migrate Up
CREATE TABLE refresh_token (
    id bigint NOT NULL,
    payload_hash bytea,
    user_id text,
    client_id text
);

CREATE SEQUENCE refresh_token_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE refresh_token_id_seq OWNED BY refresh_token.id;

ALTER TABLE ONLY refresh_token ALTER COLUMN id SET DEFAULT nextval('refresh_token_id_seq'::regclass);

ALTER TABLE ONLY refresh_token
    ADD CONSTRAINT refresh_token_pkey PRIMARY KEY (id);
