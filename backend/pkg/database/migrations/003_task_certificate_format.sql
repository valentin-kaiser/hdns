-- +migrate Up
USE hdns;

ALTER TABLE tasks
    ADD COLUMN certificate_format VARCHAR(16) NOT NULL DEFAULT 'pem' COMMENT 'certificate payload format: pem, pkcs12';

-- +migrate Down
USE hdns;

ALTER TABLE tasks
    DROP COLUMN certificate_format;
