-- name: ListFeedback :many
SELECT * FROM "Feedback"
WHERE (sqlc.narg('approved')::bool IS NULL OR "approved" = sqlc.narg('approved'))
ORDER BY "createdAt" DESC
LIMIT $1;

-- name: GetFeedback :one
SELECT * FROM "Feedback" WHERE "id" = $1;

-- name: SetFeedbackApproval :one
UPDATE "Feedback" SET
    "approved" = COALESCE(sqlc.narg('approved'), "approved"),
    "public"   = COALESCE(sqlc.narg('public'), "public"),
    "featured" = COALESCE(sqlc.narg('featured'), "featured")
WHERE "id" = $1
RETURNING *;

-- name: DeleteFeedback :execrows
DELETE FROM "Feedback" WHERE "id" = $1;

-- name: CreateFeedbackReply :one
INSERT INTO "FeedbackReply" ("feedbackId", "staffId", "message")
VALUES ($1, $2, $3)
RETURNING *;

-- name: FeedbackReplies :many
SELECT * FROM "FeedbackReply"
WHERE "feedbackId" = $1
ORDER BY "createdAt" ASC;

-- name: ListReports :many
SELECT * FROM "Report"
WHERE (sqlc.narg('status')::text IS NULL OR "status" = sqlc.narg('status'))
ORDER BY "createdAt" DESC
LIMIT $1;

-- name: GetReport :one
SELECT * FROM "Report" WHERE "id" = $1;

-- name: SetReportStatus :one
UPDATE "Report" SET "status" = $2 WHERE "id" = $1 RETURNING *;

-- name: CreateReportReply :one
INSERT INTO "ReportReply" ("reportId", "staffId", "message")
VALUES ($1, $2, $3)
RETURNING *;

-- name: ReportReplies :many
SELECT * FROM "ReportReply"
WHERE "reportId" = $1
ORDER BY "createdAt" ASC;
