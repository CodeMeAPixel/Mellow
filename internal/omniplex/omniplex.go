package omniplex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const defaultBaseURL = "https://spider.omniplex.gg"

type Client struct {
	base  string
	token string
	botID string
	http  *http.Client
}

func New(baseURL, token, botID string) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{base: baseURL, token: token, botID: botID, http: &http.Client{Timeout: 15 * time.Second}}
}

func (c *Client) Enabled() bool { return c != nil && c.token != "" && c.botID != "" }

func (c *Client) do(ctx context.Context, method, path string, body any, out any) error {
	var r io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Bot-Auth", c.token)
	req.Header.Set("User-Agent", "mellow-go")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("omniplex %s %s: %s %s", method, path, resp.Status, string(b))
	}
	if out != nil {
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return nil
}

type Stats struct {
	Servers   *int   `json:"servers,omitempty"`
	Shards    *int   `json:"shards,omitempty"`
	Users     *int   `json:"users,omitempty"`
	ShardList []int  `json:"shard_list,omitempty"`
	Status    string `json:"status,omitempty"`
}

func (c *Client) PostStats(ctx context.Context, s Stats) error {
	return c.do(ctx, http.MethodPost, "/bots/stats", s, nil)
}

type Command struct {
	Category    string `json:"category,omitempty"`
	Description string `json:"description"`
	Name        string `json:"name"`
	Usage       string `json:"usage,omitempty"`
}

func (c *Client) GetCommands(ctx context.Context) ([]Command, error) {
	var wrap struct {
		Commands []Command `json:"commands"`
	}
	if err := c.do(ctx, http.MethodGet, "/bots/"+c.botID+"/commands", nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Commands, nil
}

func (c *Client) PutCommands(ctx context.Context, cmds []Command) error {
	return c.do(ctx, http.MethodPut, "/bots/"+c.botID+"/commands", map[string]any{"commands": cmds}, nil)
}

type Changelog struct {
	ID        string `json:"id"`
	Version   string `json:"version"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
	CreatedBy string `json:"created_by"`
}

func (c *Client) GetChangelogs(ctx context.Context) ([]Changelog, error) {
	var wrap struct {
		Changelogs []Changelog `json:"changelogs"`
	}
	if err := c.do(ctx, http.MethodGet, "/bots/"+c.botID+"/changelogs", nil, &wrap); err != nil {
		return nil, err
	}
	return wrap.Changelogs, nil
}

func (c *Client) CreateChangelog(ctx context.Context, version, title, content string) (Changelog, error) {
	var out Changelog
	err := c.do(ctx, http.MethodPost, "/bots/"+c.botID+"/changelogs",
		map[string]string{"version": version, "title": title, "content": content}, &out)
	return out, err
}

func (c *Client) DeleteChangelog(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/bots/"+c.botID+"/changelogs/"+id, nil, nil)
}
