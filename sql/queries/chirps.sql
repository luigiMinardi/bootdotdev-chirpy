-- name: CreateChirp :one
INSERT INTO chirps(
    id,
    created_at,
    updated_at,
    body,
    user_id
) VALUES (
    gen_random_uuid(),
    NOW(),
    NOW(),
    $1,
    $2
)
RETURNING *;

-- name: DeleteAllChirps :exec
TRUNCATE chirps CASCADE;

-- name: DeleteChirp :one

DELETE FROM chirps
WHERE user_id = $1 AND id = $2
RETURNING *;

-- name: GetAllChirps :many
SELECT * FROM chirps
ORDER BY 
CASE WHEN UPPER(@sort_order::text) = 'ASC' THEN created_at END ASC,
CASE WHEN UPPER(@sort_order::text) = 'DESC' THEN created_at END DESC;

-- name: GetAllChirpsFromUser :many
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY 
CASE WHEN UPPER(@sort_order::text) = 'ASC' THEN created_at END ASC,
CASE WHEN UPPER(@sort_order::text) = 'DESC' THEN created_at END DESC;

-- name: GetChirp :one
SELECT * FROM chirps
WHERE id = $1 LIMIT 1;
