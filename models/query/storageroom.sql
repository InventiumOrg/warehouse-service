-- name: CreateStorageRoom :one
INSERT INTO storage_room (
    name, number, warehouse_id
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: UpdateStorageRoom :one
UPDATE storage_room
SET name = $2,
    number = $3,
    warehouse_id= $4
WHERE id = $1
RETURNING *;

-- name: GetStorageRoom :one
SELECT * FROM storage_room
WHERE id = $1;

-- name: ListStorageRoom :many
SELECT id, name, number, warehouse_id
FROM storage_room
LIMIT $1 OFFSET $2;

-- name: DeleteStorageRoom :exec
DELETE FROM storage_room
WHERE id = $1;
