package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/CodeMeAPixel/Mellow/internal/config"
)

func TestPreflightForMutatingRoutes(t *testing.T) {
	s := New(&config.Config{
		DiscordClientSecret: "x",
		ClientID:            "y",
		AllowedOrigins:      []string{"https://mymellow.xyz"},
	}, nil, nil, Directory{}, nil)
	r := chi.NewRouter()
	r.Route("/v1", s.Mount)

	for _, path := range []string{"/v1/me/preferences", "/v1/guilds/1", "/v1/me/safety-plan", "/v1/me/delete"} {
		req := httptest.NewRequest(http.MethodOptions, path, nil)
		req.Header.Set("Origin", "https://mymellow.xyz")
		req.Header.Set("Access-Control-Request-Method", "PATCH")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		if rec.Code != http.StatusNoContent {
			t.Errorf("%s: status %d, want 204", path, rec.Code)
		}
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://mymellow.xyz" {
			t.Errorf("%s: allow-origin %q", path, got)
		}
	}
}
