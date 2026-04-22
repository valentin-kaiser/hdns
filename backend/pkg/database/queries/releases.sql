-- name: GetRelease :one
SELECT
    *
FROM
    releases
WHERE
    id = ?
LIMIT
    1;

-- name: GetReleaseByTag :one
SELECT
    *
FROM
    releases
WHERE
    git_tag = ?
LIMIT
    1;

-- name: GetReleaseByCommit :one
SELECT
    *
FROM
    releases
WHERE
    git_commit = ?
LIMIT
    1;

-- name: ListReleases :many
SELECT
    *
FROM
    releases
ORDER BY
    id DESC;

-- name: CreateRelease :execlastid
INSERT INTO
    releases (
        git_tag,
        git_commit,
        git_short,
        build_date,
        go_version,
        platform
    )
VALUES
    (?, ?, ?, ?, ?, ?) ON DUPLICATE KEY
UPDATE id = id;

-- name: UpdateRelease :exec
UPDATE releases
SET
    git_tag = ?,
    git_commit = ?,
    git_short = ?,
    build_date = ?,
    go_version = ?,
    platform = ?
WHERE
    id = ?;

-- name: DeleteRelease :exec
DELETE FROM releases
WHERE
    id = ?;

-- name: GetLatestRelease :one
SELECT
    *
FROM
    releases
ORDER BY
    id DESC
LIMIT
    1;