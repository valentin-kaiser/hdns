-- name: GetRecord :one
SELECT
    *
FROM
    records
WHERE
    id = ?
LIMIT
    1;

-- name: ListRecords :many
SELECT
    r.*
FROM
    records as r
WHERE
    CASE
        WHEN sqlc.narg ('search') IS NOT NULL THEN r.name LIKE CONCAT('%', sqlc.narg ('search'), '%')
        OR r.domain LIKE CONCAT('%', sqlc.narg ('search'), '%')
        ELSE TRUE
    END
ORDER BY
    r.domain ASC,
    r.name ASC;

-- name: CreateRecord :execlastid
INSERT INTO
    records (token, zone_id, domain, name, ttl)
VALUES
    (?, ?, ?, ?, ?);

-- name: UpdateRecord :exec
UPDATE records
SET
    token = ?,
    zone_id = ?,
    domain = ?,
    name = ?,
    ttl = ?
WHERE
    id = ?;

-- name: UpdateRecordAddress :exec
UPDATE records
SET
    address_id = ?
WHERE
    id = ?;

-- name: DeleteRecord :exec
DELETE FROM records
WHERE
    id = ?;