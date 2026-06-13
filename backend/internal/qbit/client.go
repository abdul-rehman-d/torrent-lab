package qbit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Client struct {
	baseURL  string
	username string
	password string
	savePath string
	http     *http.Client
	mu       sync.Mutex
	loggedIn bool
}

type Torrent struct {
	Hash         string  `json:"hash"`
	Name         string  `json:"name"`
	Progress     float64 `json:"progress"`
	ETA          int64   `json:"eta"`
	State        string  `json:"state"`
	SavePath     string  `json:"save_path"`
	ContentPath  string  `json:"content_path"`
	CompletionOn int64   `json:"completion_on"`
}

func NewClient(baseURL, username, password, savePath string) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		username: username,
		password: password,
		savePath: savePath,
		http: &http.Client{
			Jar:     jar,
			Timeout: 20 * time.Second,
		},
	}
}

func (c *Client) ListTorrents(ctx context.Context) ([]Torrent, error) {
	req, err := c.request(ctx, http.MethodGet, "/api/v2/torrents/info", nil, "")
	if err != nil {
		return nil, err
	}

	var torrents []Torrent
	if err := c.doJSON(req, &torrents); err != nil {
		return nil, err
	}
	return torrents, nil
}

func (c *Client) AddMagnet(ctx context.Context, magnet string) error {
	form := url.Values{}
	form.Set("urls", magnet)
	if c.savePath != "" {
		form.Set("savepath", c.savePath)
	}
	req, err := c.request(ctx, http.MethodPost, "/api/v2/torrents/add", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	return c.do(req)
}

func (c *Client) AddTorrentFile(ctx context.Context, filename string, r io.Reader) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("torrents", filepath.Base(filename))
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, r); err != nil {
		return err
	}
	if c.savePath != "" {
		if err := writer.WriteField("savepath", c.savePath); err != nil {
			return err
		}
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := c.request(ctx, http.MethodPost, "/api/v2/torrents/add", &body, writer.FormDataContentType())
	if err != nil {
		return err
	}
	return c.do(req)
}

func (c *Client) DeleteTorrent(ctx context.Context, hash string, deleteFiles bool) error {
	form := url.Values{}
	form.Set("hashes", hash)
	form.Set("deleteFiles", fmt.Sprintf("%t", deleteFiles))
	req, err := c.request(ctx, http.MethodPost, "/api/v2/torrents/delete", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	return c.do(req)
}

func (c *Client) SetLocation(ctx context.Context, hash, location string) error {
	form := url.Values{}
	form.Set("hashes", hash)
	form.Set("location", location)
	req, err := c.request(ctx, http.MethodPost, "/api/v2/torrents/setLocation", strings.NewReader(form.Encode()), "application/x-www-form-urlencoded")
	if err != nil {
		return err
	}
	return c.do(req)
}

func (c *Client) login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loggedIn {
		return nil
	}

	form := url.Values{}
	form.Set("username", c.username)
	form.Set("password", c.password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		c.loggedIn = true
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "Ok." {
		return fmt.Errorf("qbit login failed: status %d body %q", resp.StatusCode, string(body))
	}
	c.loggedIn = true
	return nil
}

func (c *Client) request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Request, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req, nil
}

func (c *Client) doJSON(req *http.Request, out any) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		c.markLoggedOut()
		return fmt.Errorf("qbit request forbidden")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qbit request failed: status %d body %q", resp.StatusCode, string(body))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) do(req *http.Request) error {
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		c.markLoggedOut()
		return fmt.Errorf("qbit request forbidden")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("qbit request failed: status %d body %q", resp.StatusCode, string(body))
	}
	return nil
}

func (c *Client) markLoggedOut() {
	c.mu.Lock()
	c.loggedIn = false
	c.mu.Unlock()
}
