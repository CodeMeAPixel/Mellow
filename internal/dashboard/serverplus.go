package dashboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
)

const (
	maxExtraAlertChannels = 3
	minActivityCount      = 5
)

type serverPlusPatch struct {
	PromptEnabled        *bool     `json:"promptEnabled"`
	PromptCadence        *string   `json:"promptCadence"`
	PromptDay            *int32    `json:"promptDay"`
	PromptHour           *int32    `json:"promptHour"`
	PromptTimezone       *string   `json:"promptTimezone"`
	ExtraAlertChannelIDs *[]string `json:"extraAlertChannelIds"`
}

func (s *Service) serverStoreURL() string {
	if s.cfg.ServerPlusStoreURL != "" {
		return s.cfg.ServerPlusStoreURL
	}
	if s.billing != nil && s.billing.ServerEnabled() {
		return fmt.Sprintf("https://discord.com/discovery/applications/%s/store/%d", s.cfg.ClientID, s.billing.ServerSKU())
	}
	return ""
}

func (s *Service) hasServerPlus(ctx context.Context, guildID int64) bool {
	return s.billing != nil && s.billing.HasPlusGuild(ctx, guildID)
}

func cadenceOf(g gen.Guild) string {
	if g.PromptDay == nil {
		return "daily"
	}
	return "weekly"
}

func (s *Service) serverPlusView(ctx context.Context, g gen.Guild) map[string]any {
	day := int32(1)
	if g.PromptDay != nil {
		day = *g.PromptDay
	}
	extra := g.ExtraAlertChannelIds
	if extra == nil {
		extra = []string{}
	}
	return map[string]any{
		"available":            s.billing != nil && s.billing.ServerEnabled(),
		"active":               s.hasServerPlus(ctx, g.ID),
		"storeUrl":             s.serverStoreURL(),
		"promptEnabled":        g.PromptEnabled,
		"promptCadence":        cadenceOf(g),
		"promptDay":            day,
		"promptHour":           g.PromptHour,
		"promptTimezone":       deref(g.PromptTimezone, ""),
		"extraAlertChannelIds": extra,
	}
}

// applyServerPlus validates and saves the Server Plus settings for a server,
// writing an error response and returning false if anything is rejected.
// Turning features off is always allowed, so a lapsed server can wind them down.
func (s *Service) applyServerPlus(w http.ResponseWriter, r *http.Request, id int64, in *serverPlusPatch) bool {
	ctx := r.Context()
	g, err := s.store.GetGuild(ctx, id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "no settings saved for this server yet")
		return false
	}
	active := s.hasServerPlus(ctx, id)

	enabled := g.PromptEnabled
	if in.PromptEnabled != nil {
		enabled = *in.PromptEnabled
	}
	changesPrompt := in.PromptCadence != nil || in.PromptDay != nil || in.PromptHour != nil || in.PromptTimezone != nil
	if !active && (changesPrompt || (in.PromptEnabled != nil && *in.PromptEnabled)) {
		writeErr(w, http.StatusForbidden, "scheduled prompts are part of Server Plus")
		return false
	}

	if in.PromptEnabled != nil || changesPrompt {
		cadence := cadenceOf(g)
		if in.PromptCadence != nil {
			cadence = *in.PromptCadence
		}
		if cadence != "daily" && cadence != "weekly" {
			writeErr(w, http.StatusBadRequest, "cadence must be daily or weekly")
			return false
		}
		var day *int32
		if cadence == "weekly" {
			d := int32(1)
			if g.PromptDay != nil {
				d = *g.PromptDay
			}
			if in.PromptDay != nil {
				d = *in.PromptDay
			}
			if d < 0 || d > 6 {
				writeErr(w, http.StatusBadRequest, "day must be between 0 (Sunday) and 6 (Saturday)")
				return false
			}
			day = &d
		}
		hour := g.PromptHour
		if in.PromptHour != nil {
			hour = *in.PromptHour
		}
		if hour < 0 || hour > 23 {
			writeErr(w, http.StatusBadRequest, "hour must be between 0 and 23")
			return false
		}
		tz := g.PromptTimezone
		if in.PromptTimezone != nil {
			if *in.PromptTimezone == "" {
				tz = nil
			} else if _, err := time.LoadLocation(*in.PromptTimezone); err != nil {
				writeErr(w, http.StatusBadRequest, "invalid IANA timezone")
				return false
			} else {
				tz = in.PromptTimezone
			}
		}
		if enabled && (g.CheckInChannelId == nil || *g.CheckInChannelId == "") {
			writeErr(w, http.StatusBadRequest, "choose a check-in channel before turning prompts on")
			return false
		}
		if err := s.store.SetGuildSchedule(ctx, id, enabled, day, hour, tz); err != nil {
			writeErr(w, http.StatusInternalServerError, "could not save the prompt schedule")
			return false
		}
	}

	if in.ExtraAlertChannelIDs != nil {
		var clean []string
		seen := map[string]bool{}
		for _, ch := range *in.ExtraAlertChannelIDs {
			if ch == "" || seen[ch] {
				continue
			}
			seen[ch] = true
			clean = append(clean, ch)
		}
		if len(clean) > maxExtraAlertChannels {
			writeErr(w, http.StatusBadRequest, "you can add up to 3 extra alert channels")
			return false
		}
		if len(clean) > 0 && !active {
			writeErr(w, http.StatusForbidden, "extra alert channels are part of Server Plus")
			return false
		}
		known := map[string]bool{}
		if s.dir.Channels != nil {
			for _, c := range s.dir.Channels(id) {
				known[c.ID] = true
			}
		}
		for _, ch := range clean {
			if !known[ch] {
				writeErr(w, http.StatusBadRequest, "unknown channel")
				return false
			}
		}
		if err := s.store.SetExtraAlertChannels(ctx, id, clean); err != nil {
			writeErr(w, http.StatusInternalServerError, "could not save alert channels")
			return false
		}
	}
	return true
}

type activityWeek struct {
	Week     time.Time `json:"week"`
	CheckIns *int64    `json:"checkIns"`
	ToolUses *int64    `json:"toolUses"`
}

// suppress hides very small counts (returned as null) so activity in a tiny
// server cannot point at an individual.
func suppress(n int64) *int64 {
	if n > 0 && n < minActivityCount {
		return nil
	}
	return &n
}

func (s *Service) handleGuildActivity(w http.ResponseWriter, r *http.Request) {
	id, ok := s.guildFromRequest(w, r)
	if !ok {
		return
	}
	if !s.hasServerPlus(r.Context(), id) {
		writeErr(w, http.StatusForbidden, "the activity overview is part of Server Plus")
		return
	}
	weeks := 8
	if v, err := strconv.Atoi(r.URL.Query().Get("weeks")); err == nil {
		weeks = min(max(v, 2), 26)
	}
	rows, err := s.store.GuildActivity(r.Context(), id, weeks)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeErr(w, http.StatusInternalServerError, "could not load activity")
		return
	}
	out := make([]activityWeek, 0, len(rows))
	for _, row := range rows {
		out = append(out, activityWeek{Week: row.Week, CheckIns: suppress(row.CheckIns), ToolUses: suppress(row.ToolUses)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"weeks": weeks, "minCount": minActivityCount, "activity": out})
}
