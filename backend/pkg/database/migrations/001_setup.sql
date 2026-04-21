-- +migrate Up
-- HDNS Database Schema for MariaDB
-- This schema contains HDNS specific tables
CREATE DATABASE IF NOT EXISTS hdns CHARACTER
SET
    utf8mb4 COLLATE utf8mb4_unicode_ci;

USE hdns;

CREATE TABLE
    IF NOT EXISTS releases (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        git_tag VARCHAR(255) NOT NULL,
        git_commit VARCHAR(255) NOT NULL,
        git_short VARCHAR(255),
        build_date VARCHAR(255),
        go_version VARCHAR(255),
        platform VARCHAR(255),
        UNIQUE KEY unique_version (git_tag, git_commit, go_version, platform)
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE
    IF NOT EXISTS addresses (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        ipv4 VARCHAR(15) NULL,
        ipv6 VARCHAR(39) NULL,
        current BOOLEAN NOT NULL DEFAULT FALSE,
        UNIQUE KEY unique_address (ipv4, ipv6),
        INDEX idx_current (current)
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE
    IF NOT EXISTS records (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        token VARCHAR(255) NOT NULL,
        zone_id VARCHAR(255) NOT NULL,
        domain VARCHAR(255) NOT NULL,
        name VARCHAR(255) NOT NULL,
        ttl INT NOT NULL,
        address_id BIGINT NULL,
        last_refresh TIMESTAMP NULL,
        UNIQUE KEY unique_record (zone_id, name),
        INDEX idx_token (token),
        INDEX idx_zone_id (zone_id),
        FOREIGN KEY (address_id) REFERENCES addresses (id) ON DELETE SET NULL
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- Existing zone_id values are UUID strings from the old dns.hetzner.com API
-- and are incompatible with the new api.hetzner.cloud int64 zone IDs.
-- Clear all records to force re-setup via the UI.
UPDATE records SET zone_id = '', address_id = NULL, last_refresh = NULL;

-- +migrate Down
DROP TABLE IF EXISTS releases;
DROP TABLE IF EXISTS records;
DROP TABLE IF EXISTS addresses;

DROP DATABASE IF EXISTS hdns;