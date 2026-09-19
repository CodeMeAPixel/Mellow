package dashboard

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/CodeMeAPixel/Mellow/internal/ai"
	"github.com/CodeMeAPixel/Mellow/internal/billing"
	"github.com/CodeMeAPixel/Mellow/internal/config"
	"github.com/CodeMeAPixel/Mellow/internal/db"
	"github.com/CodeMeAPixel/Mellow/internal/db/gen"
	"github.com/CodeMeAPixel/Mellow/internal/helplines"
	"github.com/go-chi/chi/v5"
)

const (
	sessionCookie = "mellow_session"
	stateCookie   = "mellow_oauth"
	sessionTTL    = 14 * 24 * time.Hour
	discordAPI    = "https://discord.com/api/v10"

	permAdministrator = 0x8
	permManageGuild   = 0x20
)

type Channel struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Directory struct {
	HasGuild func(id int64) bool
	Channels func(id int64) []Channel
	Roles    func(id int64) []Role
}

type CommandInfo struct {
	Name            string          `json:"name"`
	Description     string          `json:"description"`
	Category        string          `json:"category"`
	GuildOnly       bool            `json:"guildOnly"`
	CooldownSeconds int             `json:"cooldownSeconds"`
	PlusCooldownSec int             `json:"plusCooldownSeconds,omitempty"`
	Permissions     []string        `json:"permissions"`
	RequiredRoles   []string        `json:"requiredRoles,omitempty"`
	Options         json.RawMessage `json:"options,omitempty"`
}

type Service struct {
	cfg      *config.Config
	store    *db.Store
	billing  *billing.Service
	dir      Directory
	commands func() []CommandInfo
	http     *http.Client
}

func New(cfg *config.Config, store *db.Store, b *billing.Service, dir Directory, commands func() []CommandInfo) *Service {
	return &Service{cfg: cfg, store: store, billing: b, dir: dir, commands: commands, http: &http.Client{Timeout: 10 * time.Second}}
}

func (s *Service) Enabled() bool { return s.cfg.DiscordClientSecret != "" && s.cfg.ClientID != "" }

func (s *Service) Mount(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(s.cors(true))
		r.Get("/commands", s.handleCommands)
		r.Options("/commands", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	})

	if !s.Enabled() {
		slog.Info("dashboard auth disabled (DISCORD_CLIENT_SECRET not set)")
		return
	}

	r.Route("/auth", func(r chi.Router) {
		r.Get("/login", s.handleLogin)
		r.Get("/callback", s.handleCallback)
	})

	r.Group(func(r chi.Router) {
		r.Use(s.cors(false))
		r.Use(s.originGuard)
		r.Options("/*", func(http.ResponseWriter, *http.Request) {})
		r.Post("/auth/logout", s.handleLogout)

		r.Group(func(r chi.Router) {
			r.Use(s.requireSession)
			r.Get("/me", s.handleMe)
			r.Get("/me/preferences", s.handleGetPrefs)
			r.Patch("/me/preferences", s.handlePatchPrefs)
			r.Get("/me/safety-plan", s.handleGetSafetyPlan)
			r.Put("/me/safety-plan", s.handlePutSafetyPlan)
			r.Get("/me/mood", s.handleMood)
			r.Get("/me/export", s.handleExport)
			r.Post("/me/delete", s.handleDeleteData)
			r.Get("/me/guilds", s.handleMyGuilds)
			r.Get("/guilds/{id}", s.handleGetGuild)
			r.Get("/guilds/{id}/activity", s.handleGuildActivity)
			r.Patch("/guilds/{id}", s.handlePatchGuild)
		})
	})

	if s.cfg.CookieSameSite == "none" {
		slog.Warn("COOKIE_SAMESITE=none: session cookies are sent cross-site, use only on a dev API")
	}
	go s.sweepSessions()
}

func (s *Service) sweepSessions() {
	for range time.Tick(time.Hour) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := s.store.DeleteExpiredWebSessions(ctx); err != nil {
			slog.Warn("session sweep failed", slog.String("err", err.Error()))
		}
		cancel()
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Service) allowedOrigin(origin string) bool {
	return origin != "" && slices.Contains(s.cfg.AllowedOrigins, origin)
}

func (s *Service) cors(public bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			h := w.Header()
			h.Add("Vary", "Origin")
			switch {
			case public:
				h.Set("Access-Control-Allow-Origin", "*")
			case s.allowedOrigin(origin):
				h.Set("Access-Control-Allow-Origin", "*")
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if r.Method == http.MethodOptions {
				h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
				h.Set("Access-Control-Allow-Headers", "Content-Type")
				h.Set("Access-Control-Max-Age", "600")
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Service) originGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if !s.allowedOrigin(r.Header.Get("Origin")) {
				writeErr(w, http.StatusForbidden, "origin not allowed")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func (s *Service) secure() bool { return strings.HasPrefix(s.cfg.APIPublicURL, "https://") }

func (s *Service) sameSite() http.SameSite {
	switch s.cfg.CookieSameSite {
	case "none":
		return http.SameSiteNoneMode
	case "strict":
		return http.SameSiteStrictMode
	}
	return http.SameSiteLaxMode
}

func (s *Service) setCookie(w http.ResponseWriter, name, value, path string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		Domain:   s.cfg.CookieDomain,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   s.secure() || s.cfg.CookieSameSite == "none",
		SameSite: s.sameSite(),
	})
}

func (s *Service) redirectURI() string { return s.cfg.APIPublicURL + "/v1/auth/callback" }

func (s *Service) handleLogin(w http.ResponseWriter, r *http.Request) {
	state, err := randomToken(16)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not start login")
		return
	}
	s.setCookie(w, stateCookie, state, "/v1/auth", 600)
	q := url.Values{
		"client_id":     {s.cfg.ClientID},
		"response_type": {"code"},
		"redirect_uri":  {s.redirectURI()},
		"scope":         {"identify guilds"},
		"state":         {state},
		"prompt":        {"none"},
	}
	http.Redirect(w, r, "https://discord.com/oauth2/authorize?"+q.Encode(), http.StatusFound)
}

func (s *Service) loginFailed(w http.ResponseWriter, r *http.Request, reason string) {
	http.Redirect(w, r, s.cfg.DashboardURL+"?error="+url.QueryEscape(reason), http.StatusFound)
}

type discordUser struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	GlobalName *string `json:"global_name"`
	Avatar     *string `json:"avatar"`
}

type discordGuild struct {
	ID          string `json:"id"`
	Owner       bool   `json:"owner"`
	Permissions string `json:"permissions"`
}

func (s *Service) discordGet(ctx context.Context, token, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discordAPI+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := s.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("discord %s returned %d", path, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (s *Service) exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{
		"client_id":     {s.cfg.ClientID},
		"client_secret": {s.cfg.DiscordClientSecret},
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {s.redirectURI()},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, discordAPI+"/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange returned %d", resp.StatusCode)
	}
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil || tok.AccessToken == "" {
		return "", errors.New("token exchange returned no token")
	}
	return tok.AccessToken, nil
}

func (s *Service) handleCallback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sc, err := r.Cookie(stateCookie)
	s.setCookie(w, stateCookie, "", "/v1/auth", -1)
	if err != nil || sc.Value == "" || sc.Value != r.URL.Query().Get("state") {
		s.loginFailed(w, r, "invalid_state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		s.loginFailed(w, r, "access_denied")
		return
	}

	token, err := s.exchange(ctx, code)
	if err != nil {
		slog.Warn("dashboard login: token exchange", slog.String("err", err.Error()))
		s.loginFailed(w, r, "login_failed")
		return
	}
	var du discordUser
	var dg []discordGuild
	if err := s.discordGet(ctx, token, "/users/@me", &du); err != nil {
		slog.Warn("dashboard login: fetch user", slog.String("err", err.Error()))
		s.loginFailed(w, r, "login_failed")
		return
	}
	if err := s.discordGet(ctx, token, "/users/@me/guilds", &dg); err != nil {
		slog.Warn("dashboard login: fetch guilds", slog.String("err", err.Error()))
		s.loginFailed(w, r, "login_failed")
		return
	}

	uid, err := strconv.ParseInt(du.ID, 10, 64)
	if err != nil {
		s.loginFailed(w, r, "login_failed")
		return
	}
	name := du.Username
	if du.GlobalName != nil && *du.GlobalName != "" {
		name = *du.GlobalName
	}
	u, err := s.store.UpsertUser(ctx, uid, name)
	if err != nil {
		slog.Warn("dashboard login: upsert user", slog.String("err", err.Error()))
		s.loginFailed(w, r, "login_failed")
		return
	}
	if u.IsBanned {
		s.loginFailed(w, r, "banned")
		return
	}

	var manageable []int64
	for _, g := range dg {
		gid, err := strconv.ParseInt(g.ID, 10, 64)
		if err != nil {
			continue
		}
		perms, _ := strconv.ParseUint(g.Permissions, 10, 64)
		if !(g.Owner || perms&permAdministrator != 0 || perms&permManageGuild != 0) {
			continue
		}
		if s.dir.HasGuild != nil && s.dir.HasGuild(gid) {
			manageable = append(manageable, gid)
		}
	}

	raw, err := randomToken(32)
	if err != nil {
		s.loginFailed(w, r, "login_failed")
		return
	}
	err = s.store.CreateWebSession(ctx, hashToken(raw), db.WebSession{
		UserID:    uid,
		Username:  name,
		Avatar:    du.Avatar,
		GuildIDs:  manageable,
		ExpiresAt: time.Now().Add(sessionTTL),
	})
	if err != nil {
		slog.Warn("dashboard login: create session", slog.String("err", err.Error()))
		s.loginFailed(w, r, "login_failed")
		return
	}
	s.setCookie(w, sessionCookie, raw, "/", int(sessionTTL.Seconds()))
	http.Redirect(w, r, s.cfg.DashboardURL, http.StatusFound)
}

type ctxKey struct{}

type session struct {
	db.WebSession
}

func (s *Service) requireSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || c.Value == "" {
			writeErr(w, http.StatusUnauthorized, "not signed in")
			return
		}
		ws, err := s.store.GetWebSession(r.Context(), hashToken(c.Value))
		if err != nil {
			writeErr(w, http.StatusUnauthorized, "session expired")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, session{ws})))
	})
}

func sess(r *http.Request) session { return r.Context().Value(ctxKey{}).(session) }

func (s *Service) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteWebSession(r.Context(), hashToken(c.Value))
	}
	s.setCookie(w, sessionCookie, "", "/", -1)
	w.WriteHeader(http.StatusNoContent)
}

type plusInfo struct {
	Active   bool       `json:"active"`
	EndsAt   *time.Time `json:"endsAt"`
	Renews   bool       `json:"renews"`
	StoreURL string     `json:"storeUrl,omitempty"`
}

func (s *Service) storeURL() string {
	if s.cfg.PlusStoreURL != "" {
		return s.cfg.PlusStoreURL
	}
	if s.billing != nil && s.billing.Enabled() {
		return fmt.Sprintf("https://discord.com/discovery/applications/%s/store/%d", s.cfg.ClientID, s.billing.PlusSKU())
	}
	return ""
}

func (s *Service) userPlus(ctx context.Context, uid int64) plusInfo {
	info := plusInfo{StoreURL: s.storeURL()}
	if s.billing == nil || !s.billing.Enabled() {
		return info
	}
	e, err := s.store.ActiveUserEntitlement(ctx, uid, int64(s.billing.PlusSKU()))
	if err != nil {
		return info
	}
	info.Active = true
	info.EndsAt = e.EndsAt
	info.Renews = e.SubscriptionId != nil
	return info
}

func avatarURL(uid int64, hash *string) string {
	if hash == nil || *hash == "" {
		return fmt.Sprintf("https://cdn.discordapp.com/embed/avatars/%d.png", (uid>>22)%6)
	}
	return fmt.Sprintf("https://cdn.discordapp.com/avatars/%d/%s.png?size=128", uid, *hash)
}

func (s *Service) handleMe(w http.ResponseWriter, r *http.Request) {
	ss := sess(r)
	writeJSON(w, http.StatusOK, map[string]any{
		"id":       strconv.FormatInt(ss.UserID, 10),
		"username": ss.Username,
		"avatar":   avatarURL(ss.UserID, ss.Avatar),
		"plus":     s.userPlus(r.Context(), ss.UserID),
	})
}

var personalities = []string{"gentle", "supportive", "direct", "playful", "professional", "encouraging"}

type prefsDTO struct {
	AIPersonality           string `json:"aiPersonality"`
	Timezone                string `json:"timezone"`
	Country                 string `json:"country"`
	Language                string `json:"language"`
	CheckInInterval         int32  `json:"checkInInterval"`
	RemindersEnabled        bool   `json:"remindersEnabled"`
	JournalPrivacy          bool   `json:"journalPrivacy"`
	DisableContextLogging   bool   `json:"disableContextLogging"`
	DisableCrisisDetection  bool   `json:"disableCrisisDetection"`
	DisableCrisisSupportDMs bool   `json:"disableCrisisSupportDMs"`
	WeeklyRecap             bool   `json:"weeklyRecap"`
	DailyPrompt             bool   `json:"dailyPrompt"`
	QuietStart              *int32 `json:"quietStart"`
	QuietEnd                *int32 `json:"quietEnd"`
	CustomPersona           string `json:"customPersona"`
}

func deref(p *string, def string) string {
	if p == nil || *p == "" {
		return def
	}
	return *p
}

func prefsToDTO(p gen.UserPreferences) prefsDTO {
	return prefsDTO{
		AIPersonality:           deref(p.AiPersonality, "gentle"),
		Timezone:                deref(p.Timezone, ""),
		Country:                 deref(p.Country, ""),
		Language:                deref(p.Language, "en"),
		CheckInInterval:         p.CheckInInterval,
		RemindersEnabled:        p.RemindersEnabled,
		JournalPrivacy:          p.JournalPrivacy,
		DisableContextLogging:   p.DisableContextLogging,
		DisableCrisisDetection:  p.DisableCrisisDetection,
		DisableCrisisSupportDMs: p.DisableCrisisSupportDMs,
		WeeklyRecap:             p.WeeklyRecap,
		DailyPrompt:             p.DailyPrompt,
		QuietStart:              p.QuietStart,
		QuietEnd:                p.QuietEnd,
		CustomPersona:           deref(p.CustomPersona, ""),
	}
}

func (s *Service) hasPlus(ctx context.Context, uid int64) bool {
	if s.billing == nil || !s.billing.Enabled() {
		return false
	}
	has, _ := s.billing.HasPlusUser(ctx, uid)
	return has
}

func (s *Service) handleGetPrefs(w http.ResponseWriter, r *http.Request) {
	p, err := s.store.EnsureUserPreferences(r.Context(), sess(r).UserID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not load preferences")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"preferences":       prefsToDTO(p),
		"personalities":     personalities,
		"plusPersonalities": ai.PlusPersonalities,
		"plus":              s.hasPlus(r.Context(), sess(r).UserID),
		"countries":         helplines.Countries(),
	})
}

type prefsPatch struct {
	AIPersonality           *string `json:"aiPersonality"`
	Timezone                *string `json:"timezone"`
	Country                 *string `json:"country"`
	Language                *string `json:"language"`
	CheckInInterval         *int32  `json:"checkInInterval"`
	RemindersEnabled        *bool   `json:"remindersEnabled"`
	JournalPrivacy          *bool   `json:"journalPrivacy"`
	DisableContextLogging   *bool   `json:"disableContextLogging"`
	DisableCrisisDetection  *bool   `json:"disableCrisisDetection"`
	DisableCrisisSupportDMs *bool   `json:"disableCrisisSupportDMs"`
	WeeklyRecap             *bool   `json:"weeklyRecap"`
	DailyPrompt             *bool   `json:"dailyPrompt"`
	QuietStart              *int32  `json:"quietStart"`
	QuietEnd                *int32  `json:"quietEnd"`
	QuietOff                *bool   `json:"quietOff"`
	CustomPersona           *string `json:"customPersona"`
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func (s *Service) handlePatchPrefs(w http.ResponseWriter, r *http.Request) {
	var in prefsPatch
	if !decode(w, r, &in) {
		return
	}
	plus := s.hasPlus(r.Context(), sess(r).UserID)
	if in.AIPersonality != nil {
		switch {
		case slices.Contains(personalities, *in.AIPersonality):
		case ai.IsPlusPersonality(*in.AIPersonality):
			if !plus {
				writeErr(w, http.StatusForbidden, "that style is part of Mellow Plus")
				return
			}
		default:
			writeErr(w, http.StatusBadRequest, "unknown personality")
			return
		}
	}
	if in.CustomPersona != nil {
		if !plus {
			writeErr(w, http.StatusForbidden, "a custom style is part of Mellow Plus")
			return
		}
		clean := ai.SanitizePersona(*in.CustomPersona)
		in.CustomPersona = &clean
	}
	quietOff := in.QuietOff != nil && *in.QuietOff
	if !quietOff && (in.QuietStart != nil || in.QuietEnd != nil) {
		if in.QuietStart == nil || in.QuietEnd == nil ||
			*in.QuietStart < 0 || *in.QuietStart > 23 || *in.QuietEnd < 0 || *in.QuietEnd > 23 {
			writeErr(w, http.StatusBadRequest, "quiet hours need a start and end between 0 and 23")
			return
		}
	}
	if in.Timezone != nil {
		if _, err := time.LoadLocation(*in.Timezone); err != nil || *in.Timezone == "" {
			writeErr(w, http.StatusBadRequest, "invalid IANA timezone")
			return
		}
	}
	if in.Language != nil && (len(*in.Language) < 2 || len(*in.Language) > 10) {
		writeErr(w, http.StatusBadRequest, "invalid language code")
		return
	}
	if in.Country != nil && *in.Country != "" {
		c, ok := helplines.Lookup(*in.Country)
		if !ok {
			writeErr(w, http.StatusBadRequest, "unsupported country")
			return
		}
		in.Country = &c.Code
	}
	if in.CheckInInterval != nil && (*in.CheckInInterval < 30 || *in.CheckInInterval > 10080) {
		writeErr(w, http.StatusBadRequest, "check-in interval must be between 30 and 10080 minutes")
		return
	}

	p, err := s.store.UpdateUserPreferences(r.Context(), sess(r).UserID, db.PrefsUpdate{
		AIPersonality:           in.AIPersonality,
		Timezone:                in.Timezone,
		Country:                 in.Country,
		Language:                in.Language,
		CheckInInterval:         in.CheckInInterval,
		RemindersEnabled:        in.RemindersEnabled,
		JournalPrivacy:          in.JournalPrivacy,
		DisableContextLogging:   in.DisableContextLogging,
		DisableCrisisDetection:  in.DisableCrisisDetection,
		DisableCrisisSupportDMs: in.DisableCrisisSupportDMs,
		WeeklyRecap:             in.WeeklyRecap,
		DailyPrompt:             in.DailyPrompt,
		CustomPersona:           in.CustomPersona,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not save preferences")
		return
	}
	if quietOff || in.QuietStart != nil {
		start, end := in.QuietStart, in.QuietEnd
		if quietOff {
			start, end = nil, nil
		}
		if err := s.store.SetQuietHours(r.Context(), sess(r).UserID, start, end); err != nil {
			writeErr(w, http.StatusInternalServerError, "could not save quiet hours")
			return
		}
		if p, err = s.store.GetUserPreferences(r.Context(), sess(r).UserID); err != nil {
			writeErr(w, http.StatusInternalServerError, "saved, but could not reload preferences")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"preferences": prefsToDTO(p)})
}

func (s *Service) handleMyGuilds(w http.ResponseWriter, r *http.Request) {
	ss := sess(r)
	type item struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		Plus bool   `json:"plus"`
	}
	out := []item{}
	for _, id := range ss.GuildIDs {
		if s.dir.HasGuild != nil && !s.dir.HasGuild(id) {
			continue
		}
		g, err := s.store.GetGuild(r.Context(), id)
		if err != nil {
			continue
		}
		out = append(out, item{ID: strconv.FormatInt(id, 10), Name: g.Name, Plus: s.hasServerPlus(r.Context(), id)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"guilds": out})
}

type guildDTO struct {
	ID                    string  `json:"id"`
	Name                  string  `json:"name"`
	SystemChannelID       *string `json:"systemChannelId"`
	ModAlertChannelID     *string `json:"modAlertChannelId"`
	ModLogChannelID       *string `json:"modLogChannelId"`
	CheckInChannelID      *string `json:"checkInChannelId"`
	ModeratorRoleID       *string `json:"moderatorRoleId"`
	EnableCheckIns        bool    `json:"enableCheckIns"`
	EnableGhostLetters    bool    `json:"enableGhostLetters"`
	EnableCrisisAlerts    bool    `json:"enableCrisisAlerts"`
	SystemLogsEnabled     bool    `json:"systemLogsEnabled"`
	DisableContextLogging bool    `json:"disableContextLogging"`
	Language              string  `json:"language"`
}

func (s *Service) guildFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid guild id")
		return 0, false
	}
	if !slices.Contains(sess(r).GuildIDs, id) {
		writeErr(w, http.StatusForbidden, "you cannot manage this server")
		return 0, false
	}
	if s.dir.HasGuild != nil && !s.dir.HasGuild(id) {
		writeErr(w, http.StatusNotFound, "Mellow is no longer in this server")
		return 0, false
	}
	return id, true
}

func (s *Service) guildResponse(ctx context.Context, id int64) (map[string]any, error) {
	g, err := s.store.GetGuild(ctx, id)
	if err != nil {
		return nil, err
	}
	dto := guildDTO{
		ID: strconv.FormatInt(g.ID, 10), Name: g.Name,
		SystemChannelID: g.SystemChannelId, ModAlertChannelID: g.ModAlertChannelId,
		ModLogChannelID: g.ModLogChannelId, CheckInChannelID: g.CheckInChannelId,
		ModeratorRoleID: g.ModeratorRoleId,
		EnableCheckIns:  g.EnableCheckIns, EnableGhostLetters: g.EnableGhostLetters,
		EnableCrisisAlerts: g.EnableCrisisAlerts, SystemLogsEnabled: g.SystemLogsEnabled,
		DisableContextLogging: g.DisableContextLogging, Language: deref(g.Language, "en"),
	}
	channels, roles := []Channel{}, []Role{}
	if s.dir.Channels != nil {
		channels = s.dir.Channels(id)
	}
	if s.dir.Roles != nil {
		roles = s.dir.Roles(id)
	}
	return map[string]any{
		"guild": dto, "channels": channels, "roles": roles,
		"serverPlus": s.serverPlusView(ctx, g),
	}, nil
}

func (s *Service) handleGetGuild(w http.ResponseWriter, r *http.Request) {
	id, ok := s.guildFromRequest(w, r)
	if !ok {
		return
	}
	resp, err := s.guildResponse(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusNotFound, "no settings saved for this server yet")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

type guildPatch struct {
	SystemChannelID       *string `json:"systemChannelId"`
	ModAlertChannelID     *string `json:"modAlertChannelId"`
	ModLogChannelID       *string `json:"modLogChannelId"`
	CheckInChannelID      *string `json:"checkInChannelId"`
	ModeratorRoleID       *string `json:"moderatorRoleId"`
	EnableCheckIns        *bool   `json:"enableCheckIns"`
	EnableGhostLetters    *bool   `json:"enableGhostLetters"`
	EnableCrisisAlerts    *bool   `json:"enableCrisisAlerts"`
	SystemLogsEnabled     *bool   `json:"systemLogsEnabled"`
	DisableContextLogging *bool   `json:"disableContextLogging"`
	Language              *string `json:"language"`

	ServerPlus *serverPlusPatch `json:"serverPlus"`
}

func (s *Service) handlePatchGuild(w http.ResponseWriter, r *http.Request) {
	id, ok := s.guildFromRequest(w, r)
	if !ok {
		return
	}
	var in guildPatch
	if !decode(w, r, &in) {
		return
	}

	channelOK := func(v *string) bool {
		if v == nil || *v == "" {
			return true
		}
		if s.dir.Channels == nil {
			return false
		}
		for _, c := range s.dir.Channels(id) {
			if c.ID == *v {
				return true
			}
		}
		return false
	}
	roleOK := func(v *string) bool {
		if v == nil || *v == "" {
			return true
		}
		if s.dir.Roles == nil {
			return false
		}
		for _, ro := range s.dir.Roles(id) {
			if ro.ID == *v {
				return true
			}
		}
		return false
	}
	if !channelOK(in.SystemChannelID) || !channelOK(in.ModAlertChannelID) || !channelOK(in.ModLogChannelID) || !channelOK(in.CheckInChannelID) {
		writeErr(w, http.StatusBadRequest, "unknown channel")
		return
	}
	if !roleOK(in.ModeratorRoleID) {
		writeErr(w, http.StatusBadRequest, "unknown role")
		return
	}
	if in.Language != nil && (len(*in.Language) < 2 || len(*in.Language) > 10) {
		writeErr(w, http.StatusBadRequest, "invalid language code")
		return
	}

	if _, err := s.store.GetGuild(r.Context(), id); err != nil {
		writeErr(w, http.StatusNotFound, "no settings saved for this server yet")
		return
	}
	_, err := s.store.UpdateGuildSettings(r.Context(), id, db.GuildSettingsUpdate{
		SystemChannelID: in.SystemChannelID, ModAlertChannelID: in.ModAlertChannelID,
		ModLogChannelID: in.ModLogChannelID, CheckInChannelID: in.CheckInChannelID,
		ModeratorRoleID: in.ModeratorRoleID, EnableCheckIns: in.EnableCheckIns,
		EnableGhostLetters: in.EnableGhostLetters, EnableCrisisAlerts: in.EnableCrisisAlerts,
		SystemLogsEnabled: in.SystemLogsEnabled, DisableContextLogging: in.DisableContextLogging,
		Language: in.Language,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not save settings")
		return
	}
	if in.ServerPlus != nil && !s.applyServerPlus(w, r, id, in.ServerPlus) {
		return
	}
	resp, err := s.guildResponse(r.Context(), id)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "saved, but could not reload settings")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Service) handleCommands(w http.ResponseWriter, r *http.Request) {
	cmds := []CommandInfo{}
	if s.commands != nil {
		cmds = s.commands()
	}
	w.Header().Set("Cache-Control", "public, max-age=300")
	writeJSON(w, http.StatusOK, map[string]any{"commands": cmds})
}
