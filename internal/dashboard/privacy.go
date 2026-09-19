package dashboard

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/CodeMeAPixel/Mellow/internal/db"
)

const (
	maxPlanField = 2000
	maxBody      = 32 << 10
)

func decodeLimited(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func (s *Service) handleGetSafetyPlan(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.GetSafetyPlan(r.Context(), sess(r).UserID)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeErr(w, http.StatusInternalServerError, "could not load your safety plan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": p})
}

type safetyPlanBody struct {
	WarningSigns     string `json:"warningSigns"`
	CopingStrategies string `json:"copingStrategies"`
	Distractions     string `json:"distractions"`
	PeopleToAsk      string `json:"peopleToAsk"`
	Professionals    string `json:"professionals"`
	SafeEnvironment  string `json:"safeEnvironment"`
}

func (s *Service) handlePutSafetyPlan(w http.ResponseWriter, r *http.Request) {
	var in safetyPlanBody
	if !decodeLimited(w, r, &in) {
		return
	}
	for _, v := range []string{in.WarningSigns, in.CopingStrategies, in.Distractions, in.PeopleToAsk, in.Professionals, in.SafeEnvironment} {
		if utf8.RuneCountInString(v) > maxPlanField {
			writeErr(w, http.StatusBadRequest, "each section must be 2000 characters or fewer")
			return
		}
	}
	plan := db.SafetyPlan{
		WarningSigns: in.WarningSigns, CopingStrategies: in.CopingStrategies, Distractions: in.Distractions,
		PeopleToAsk: in.PeopleToAsk, Professionals: in.Professionals, SafeEnvironment: in.SafeEnvironment,
	}
	uid := sess(r).UserID
	if err := s.store.SaveSafetyPlan(r.Context(), uid, plan); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not save your safety plan")
		return
	}
	saved, err := s.store.GetSafetyPlan(r.Context(), uid)
	if err != nil && !errors.Is(err, db.ErrNotFound) {
		writeErr(w, http.StatusInternalServerError, "saved, but could not reload your safety plan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"plan": saved})
}

func (s *Service) handleExport(w http.ResponseWriter, r *http.Request) {
	data, err := s.store.ExportUserData(r.Context(), sess(r).UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not build your export")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", `attachment; filename="mellow-data-`+time.Now().UTC().Format("2006-01-02")+`.json"`)
	w.Header().Set("Cache-Control", "no-store")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(data)
}

func (s *Service) handleDeleteData(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Confirm string `json:"confirm"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Confirm != "DELETE" {
		writeErr(w, http.StatusBadRequest, `type "DELETE" to confirm`)
		return
	}
	if err := s.store.EraseUserData(r.Context(), sess(r).UserID); err != nil {
		writeErr(w, http.StatusInternalServerError, "could not delete your data")
		return
	}
	s.setCookie(w, sessionCookie, "", "/", -1)
	w.WriteHeader(http.StatusNoContent)
}

type moodPoint struct {
	At        time.Time `json:"at"`
	Mood      string    `json:"mood"`
	Intensity *int32    `json:"intensity"`
}

func (s *Service) handleMood(w http.ResponseWriter, r *http.Request) {
	days := 30
	if v, err := strconv.Atoi(r.URL.Query().Get("days")); err == nil {
		days = min(max(v, 7), 365)
	}
	rows, err := s.store.MoodCheckInsSince(r.Context(), sess(r).UserID, time.Now().AddDate(0, 0, -days))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not load check-ins")
		return
	}
	points := make([]moodPoint, 0, len(rows))
	counts := map[string]int{}
	for _, m := range rows {
		points = append(points, moodPoint{At: m.CreatedAt, Mood: m.Mood, Intensity: m.Intensity})
		counts[m.Mood]++
	}
	writeJSON(w, http.StatusOK, map[string]any{"days": days, "total": len(points), "points": points, "counts": counts})
}
