-- +migrate Up
USE hdns;

ALTER TABLE records
    ADD COLUMN purpose TINYINT NOT NULL DEFAULT 1 COMMENT '1=ddns, 2=cert, 3=both',
    ADD COLUMN include_wildcard BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE
    IF NOT EXISTS certificates (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        record_id BIGINT NOT NULL,
        domains TEXT NOT NULL COMMENT 'comma-separated list of certificate SANs',
        status VARCHAR(32) NOT NULL DEFAULT 'pending' COMMENT 'pending, valid, failed, expired',
        not_before TIMESTAMP NULL,
        not_after TIMESTAMP NULL,
        serial VARCHAR(128) NULL,
        last_error TEXT NULL,
        cert_path VARCHAR(512) NOT NULL DEFAULT '',
        key_path VARCHAR(512) NOT NULL DEFAULT '',
        UNIQUE KEY unique_certificate_record (record_id),
        INDEX idx_cert_status (status),
        INDEX idx_cert_not_after (not_after),
        FOREIGN KEY (record_id) REFERENCES records (id) ON DELETE CASCADE
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE
    IF NOT EXISTS tasks (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        record_id BIGINT NOT NULL,
        name VARCHAR(255) NOT NULL,
        trigger_on TINYINT NOT NULL DEFAULT 1 COMMENT '1=ip, 2=cert, 3=both',
        method VARCHAR(8) NOT NULL DEFAULT 'POST',
        url VARCHAR(2048) NOT NULL,
        headers TEXT NULL COMMENT 'AES-256-GCM encrypted JSON object of HTTP headers (base64url-encoded nonce+ciphertext+tag)',
        body TEXT NULL,
        include_certificate BOOLEAN NOT NULL DEFAULT FALSE COMMENT 'inject the issued certificate and private key into the webhook body',
        enabled BOOLEAN NOT NULL DEFAULT TRUE,
        last_run TIMESTAMP NULL,
        last_status VARCHAR(32) NULL,
        last_error TEXT NULL,
        INDEX idx_task_record (record_id),
        FOREIGN KEY (record_id) REFERENCES records (id) ON DELETE CASCADE
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- +migrate Down
USE hdns;

DROP TABLE IF EXISTS tasks;

DROP TABLE IF EXISTS certificates;

ALTER TABLE records
    DROP COLUMN include_wildcard,
    DROP COLUMN purpose;
