-- name: GetTask :one
SELECT
    *
FROM
    tasks
WHERE
    id = ?
LIMIT
    1;

-- name: ListTasks :many
SELECT
    *
FROM
    tasks
ORDER BY
    created_at DESC;

-- name: ListTasksByRecord :many
SELECT
    *
FROM
    tasks
WHERE
    record_id = ?
ORDER BY
    created_at DESC;

-- name: ListEnabledTasksByRecord :many
SELECT
    *
FROM
    tasks
WHERE
    record_id = ?
    AND enabled = TRUE
ORDER BY
    created_at ASC;

-- name: CreateTask :execlastid
INSERT INTO
    tasks (record_id, name, trigger_on, method, url, headers, body, enabled, include_certificate)
VALUES
    (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateTask :exec
UPDATE tasks
SET
    record_id = ?,
    name = ?,
    trigger_on = ?,
    method = ?,
    url = ?,
    headers = ?,
    body = ?,
    enabled = ?,
    include_certificate = ?
WHERE
    id = ?;

-- name: UpdateTaskResult :exec
UPDATE tasks
SET
    last_run = ?,
    last_status = ?,
    last_error = ?
WHERE
    id = ?;

-- name: DeleteTask :exec
DELETE FROM tasks
WHERE
    id = ?;
