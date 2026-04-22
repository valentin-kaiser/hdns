-- name: GetCurrentAddress :one
SELECT
    *
FROM
    addresses
WHERE
    current = TRUE
LIMIT
    1;

-- name: GetAddressByID :one
SELECT
    *
FROM
    addresses
WHERE
    id = ?;

-- name: ListAddresses :many
SELECT
    *
FROM
    addresses
ORDER BY
    id DESC;

-- name: CreateAddress :execlastid
INSERT INTO
    addresses (
        ipv4,
        ipv6,
        current
    )
VALUES
    (?, ?, ?);

-- name: UpdateAddress :exec
UPDATE addresses
SET
    ipv4 = ?,
    ipv6 = ?,
    current = ?
WHERE
    id = ?;

-- name: ResetCurrentAddresses :exec
UPDATE addresses
SET
    current = FALSE
WHERE
    current = TRUE;

-- name: DeleteAddress :exec
DELETE FROM addresses
WHERE
    id = ?;