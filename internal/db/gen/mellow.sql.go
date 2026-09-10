package gen

import (
	"context"
)

const getMellow = `-- name: GetMellow :one
SELECT id, model, prompt, temperature, "presencePenalty", "frequencyPenalty", "maxTokens", enabled, "checkInTools", "copingTools", "ghostTools", "crisisTools", owners, "feedbackLogs", "reportLogs", "serverId", "adminId", "modId", "logId" FROM "Mellow" WHERE "id" = 1
`

func (q *Queries) GetMellow(ctx context.Context) (Mellow, error) {
	row := q.db.QueryRow(ctx, getMellow)
	var i Mellow
	err := row.Scan(
		&i.ID,
		&i.Model,
		&i.Prompt,
		&i.Temperature,
		&i.PresencePenalty,
		&i.FrequencyPenalty,
		&i.MaxTokens,
		&i.Enabled,
		&i.CheckInTools,
		&i.CopingTools,
		&i.GhostTools,
		&i.CrisisTools,
		&i.Owners,
		&i.FeedbackLogs,
		&i.ReportLogs,
		&i.ServerId,
		&i.AdminId,
		&i.ModId,
		&i.LogId,
	)
	return i, err
}

const setMellowEnabled = `-- name: SetMellowEnabled :exec
UPDATE "Mellow" SET "enabled" = $1 WHERE "id" = 1
`

func (q *Queries) SetMellowEnabled(ctx context.Context, enabled bool) error {
	_, err := q.db.Exec(ctx, setMellowEnabled, enabled)
	return err
}

const updateMellowAIConfig = `-- name: UpdateMellowAIConfig :one
UPDATE "Mellow" SET
    "model"       = COALESCE($1, "model"),
    "prompt"      = COALESCE($2, "prompt"),
    "temperature" = COALESCE($3, "temperature"),
    "maxTokens"   = COALESCE($4, "maxTokens")
WHERE "id" = 1
RETURNING id, model, prompt, temperature, "presencePenalty", "frequencyPenalty", "maxTokens", enabled, "checkInTools", "copingTools", "ghostTools", "crisisTools", owners, "feedbackLogs", "reportLogs", "serverId", "adminId", "modId", "logId"
`

type UpdateMellowAIConfigParams struct {
	Model       *string  `json:"model"`
	Prompt      *string  `json:"prompt"`
	Temperature *float64 `json:"temperature"`
	MaxTokens   *int32   `json:"max_tokens"`
}

func (q *Queries) UpdateMellowAIConfig(ctx context.Context, arg UpdateMellowAIConfigParams) (Mellow, error) {
	row := q.db.QueryRow(ctx, updateMellowAIConfig,
		arg.Model,
		arg.Prompt,
		arg.Temperature,
		arg.MaxTokens,
	)
	var i Mellow
	err := row.Scan(
		&i.ID,
		&i.Model,
		&i.Prompt,
		&i.Temperature,
		&i.PresencePenalty,
		&i.FrequencyPenalty,
		&i.MaxTokens,
		&i.Enabled,
		&i.CheckInTools,
		&i.CopingTools,
		&i.GhostTools,
		&i.CrisisTools,
		&i.Owners,
		&i.FeedbackLogs,
		&i.ReportLogs,
		&i.ServerId,
		&i.AdminId,
		&i.ModId,
		&i.LogId,
	)
	return i, err
}
