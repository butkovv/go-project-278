-- name: ListLinkVisits :many
SELECT
  id,
  link_id,
  created_at,
  ip,
  user_agent,
  status
FROM link_visits
ORDER BY id DESC
LIMIT $1 OFFSET $2;

-- name: CreateLinkVisit :exec
INSERT INTO link_visits (link_id, ip, user_agent, status)
VALUES ($1, $2, $3, $4);

-- name: GetTotalLinkVisitsCount :one
SELECT count(*) AS total_count FROM link_visits;
