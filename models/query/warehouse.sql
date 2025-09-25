-- name: CreateWarehouse :one
INSERT INTO warehouse (
    name, address, ward, district, city, country
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateWarehouse :one
UPDATE warehouse
SET name = $2,
    address = $3,
    ward = $4,
    district = $5,
    city = $6,
    country = $7
WHERE id = $1
RETURNING *;

-- name: GetWarehouse :one
SELECT * FROM warehouse
WHERE id = $1;

-- name: ListWarehouse :many
SELECT id, name, address, ward, district, city, country
FROM warehouse
LIMIT $1 OFFSET $2;

-- name: DeleteWarehouse :exec
DELETE FROM warehouse
WHERE id = $1;
