package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	repo  string
	token string
	http  *http.Client
}

func New(repo, token string) *Client {
	for _, p := range []string{"https://github.com/", "http://github.com/", "github.com/"} {
		if strings.HasPrefix(repo, p) {
			repo = strings.TrimSuffix(strings.TrimPrefix(repo, p), ".git")
			break
		}
	}
	if repo == "" {
		repo = "CodeMeAPixel/Mellow"
	}
	return &Client{repo: repo, token: token, http: &http.Client{Timeout: 10 * time.Second}}
}

type RepoInfo struct {
	FullName    string `json:"full_name"`
	Description string `json:"description"`
	Stars       int    `json:"stargazers_count"`
	Forks       int    `json:"forks_count"`
	OpenIssues  int    `json:"open_issues_count"`
	PushedAt    string `json:"pushed_at"`
	HTMLURL     string `json:"html_url"`
}

type Release struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Body        string `json:"body"`
}

func (c *Client) get(ctx context.Context, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/"+c.repo+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "mellow-go")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("github api %s: %s", path, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) RepoInfo(ctx context.Context) (RepoInfo, error) {
	var r RepoInfo
	err := c.get(ctx, "", &r)
	return r, err
}

func (c *Client) LatestRelease(ctx context.Context) (Release, error) {
	var r Release
	err := c.get(ctx, "/releases/latest", &r)
	return r, err
}

func (c *Client) LatestTag(ctx context.Context) (string, error) {
	var tags []struct {
		Name string `json:"name"`
	}
	if err := c.get(ctx, "/tags?per_page=1", &tags); err != nil {
		return "", err
	}
	if len(tags) == 0 {
		return "", nil
	}
	return tags[0].Name, nil
}
