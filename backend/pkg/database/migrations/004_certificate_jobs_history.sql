-- +migrate Up
USE hdns;

CREATE TABLE
    IF NOT EXISTS certificate_jobs (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        record_id BIGINT NOT NULL,
        certificate_id BIGINT NULL,
        source VARCHAR(32) NOT NULL DEFAULT 'manual' COMMENT 'manual, scheduled',
        status VARCHAR(32) NOT NULL DEFAULT 'running' COMMENT 'running, success, failed',
        started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        finished_at TIMESTAMP NULL,
        error TEXT NULL,
        INDEX idx_certificate_jobs_record (record_id),
        INDEX idx_certificate_jobs_certificate (certificate_id),
        INDEX idx_certificate_jobs_started (started_at),
        FOREIGN KEY (record_id) REFERENCES records (id) ON DELETE CASCADE,
        FOREIGN KEY (certificate_id) REFERENCES certificates (id) ON DELETE SET NULL
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE
    IF NOT EXISTS task_runs (
        id BIGINT AUTO_INCREMENT PRIMARY KEY,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
        task_id BIGINT NOT NULL,
        record_id BIGINT NOT NULL,
        certificate_job_id BIGINT NULL,
        trigger_on TINYINT NOT NULL DEFAULT 1 COMMENT '1=ip, 2=cert, 3=both',
        status VARCHAR(32) NOT NULL DEFAULT 'failed' COMMENT 'success, failed',
        response_status VARCHAR(32) NULL,
        error TEXT NULL,
        started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        finished_at TIMESTAMP NULL,
        INDEX idx_task_runs_task (task_id),
        INDEX idx_task_runs_record (record_id),
        INDEX idx_task_runs_job (certificate_job_id),
        INDEX idx_task_runs_started (started_at),
        FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
        FOREIGN KEY (record_id) REFERENCES records (id) ON DELETE CASCADE,
        FOREIGN KEY (certificate_job_id) REFERENCES certificate_jobs (id) ON DELETE SET NULL
    ) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

-- +migrate Down
USE hdns;

DROP TABLE IF EXISTS task_runs;

DROP TABLE IF EXISTS certificate_jobs;
