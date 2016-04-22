-- +migrate Up
CREATE TABLE IF NOT EXISTS "trusted_peers" (
       "client_id" text not null,
       "trusted_client_id" text not null,
       primary key ("client_id", "trusted_client_id")) ;
