-- name: GetCertificate :one
SELECT
    *
FROM
    certificates
WHERE
    id = ?
LIMIT
    1;

-- name: GetCertificateByRecord :one
SELECT
    *
FROM
    certificates
WHERE
    record_id = ?
LIMIT
    1;

-- name: ListCertificates :many
SELECT
    *
FROM
    certificates
ORDER BY
    created_at DESC;

-- name: ListCertificatesForRenewal :many
SELECT
    *
FROM
    certificates
WHERE
    not_after IS NOT NULL
    AND not_after <= ?
ORDER BY
    not_after ASC;

-- name: CreateCertificate :execlastid
INSERT INTO
    certificates (record_id, domains, status, cert_path, key_path)
VALUES
    (?, ?, ?, ?, ?);

-- name: UpdateCertificate :exec
UPDATE certificates
SET
    domains = ?,
    status = ?,
    not_before = ?,
    not_after = ?,
    serial = ?,
    last_error = ?,
    cert_path = ?,
    key_path = ?
WHERE
    id = ?;

-- name: UpdateCertificateStatus :exec
UPDATE certificates
SET
    status = ?,
    last_error = ?
WHERE
    id = ?;

-- name: DeleteCertificate :exec
DELETE FROM certificates
WHERE
    id = ?;

-- name: DeleteCertificateByRecord :exec
DELETE FROM certificates
WHERE
    record_id = ?;
