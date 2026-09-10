package gen

import (
	"context"
	"time"
)

const countUsers = `-- name: CountUsers :one
SELECT COUNT(*) FROM "User"
`

func (q *Queries) CountUsers(ctx context.Context) (int64, error) {
	row := q.db.QueryRow(ctx, countUsers)
	var count int64
	err := row.Scan(&count)
	return count, err
}

const getUser = `-- name: GetUser :one
SELECT id, username, role, "createdAt", "isBanned", "bannedUntil", "banReason", "discordId" FROM "User" WHERE "id" = $1
`

func (q *Queries) GetUser(ctx context.Context, id int64) (User, error) {
	row := q.db.QueryRow(ctx, getUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Role,
		&i.CreatedAt,
		&i.IsBanned,
		&i.BannedUntil,
		&i.BanReason,
		&i.DiscordId,
	)
	return i, err
}

const setUserBan = `-- name: SetUserBan :exec
UPDATE "User"
SET "isBanned" = $2, "bannedUntil" = $3, "banReason" = $4
WHERE "id" = $1
`

type SetUserBanParams struct {
	ID          int64      `json:"id"`
	IsBanned    bool       `json:"isBanned"`
	BannedUntil *time.Time `json:"bannedUntil"`
	BanReason   *string    `json:"banReason"`
}

func (q *Queries) SetUserBan(ctx context.Context, arg SetUserBanParams) error {
	_, err := q.db.Exec(ctx, setUserBan,
		arg.ID,
		arg.IsBanned,
		arg.BannedUntil,
		arg.BanReason,
	)
	return err
}

const setUserRole = `-- name: SetUserRole :exec
UPDATE "User" SET "role" = $1::text::"Role" WHERE "id" = $2
`

type SetUserRoleParams struct {
	Role string `json:"role"`
	ID   int64  `json:"id"`
}

func (q *Queries) SetUserRole(ctx context.Context, arg SetUserRoleParams) error {
	_, err := q.db.Exec(ctx, setUserRole, arg.Role, arg.ID)
	return err
}

const upsertUser = `-- name: UpsertUser :one
INSERT INTO "User" ("id", "username")
VALUES ($1, $2)
ON CONFLICT ("id") DO UPDATE SET "username" = EXCLUDED."username"
RETURNING id, username, role, "createdAt", "isBanned", "bannedUntil", "banReason", "discordId"
`

type UpsertUserParams struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}

func (q *Queries) UpsertUser(ctx context.Context, arg UpsertUserParams) (User, error) {
	row := q.db.QueryRow(ctx, upsertUser, arg.ID, arg.Username)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Username,
		&i.Role,
		&i.CreatedAt,
		&i.IsBanned,
		&i.BannedUntil,
		&i.BanReason,
		&i.DiscordId,
	)
	return i, err
}
