-- +migrate Up
USE hdns;

ALTER TABLE records
    ADD COLUMN acme_email VARCHAR(255) NULL DEFAULT NULL COMMENT 'optional per-record ACME account email; falls back to global acme.email when NULL';

-- +migrate Down
USE hdns;

ALTER TABLE records DROP COLUMN acme_email;
