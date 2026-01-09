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
    *
FROM
    records
ORDER BY
    id DESC;

-- name: CreateRecord :execlastid
INSERT INTO
    records (
        token,
        zone_id,
        domain,
        name,
        ttl,
        address_id,
        last_refresh
    )
VALUES
    (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateRecord :execlastid
UPDATE records
SET
    token = ?,
    zone_id = ?,
    domain = ?,
    name = ?,
    ttl = ?,
    address_id = ?,
    last_refresh = ?
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