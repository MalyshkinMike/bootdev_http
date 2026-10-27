-- name: CreateToken :one
insert into refresh_tokens (
    token, 
    created_at, 
    updated_at, 
    user_id, 
    expires_at
)
values (
  $1,
  now(),
  now(),
  $2,
  $3
)
returning *;

-- name: GetToken :one
select * from refresh_tokens
where token = $1 limit 1;

-- name: RevokeToken :exec
update refresh_tokens
set 
  revoked_at = CURRENT_TIMESTAMP,
  updated_at = CURRENT_TIMESTAMP
where token = $1;
