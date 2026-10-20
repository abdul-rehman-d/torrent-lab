package qbit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

var ErrNotConfigured = errors.New("qBittorrent is not configured")

type Client struct {
	baseURL  string
	username string
	password string
	savePath string
	http     *http.Client
	mu       sync.Mutex
	loggedIn bool
	blocked  error
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

func (c *Client) Configure(baseURL, username, password string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.baseURL = strings.TrimRight(baseURL, "/")
	c.username = username
	c.password = password
	c.loggedIn = false
	c.blocked = nil
	jar, _ := cookiejar.New(nil)
	c.http.Jar = jar
}

func (c *Client) Restart() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loggedIn = false
	c.blocked = nil
	jar, _ := cookiejar.New(nil)
	c.http.Jar = jar
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

type TorrentFile struct {
	Name string `json:"name"`
}

func (c *Client) ListTorrentFiles(ctx context.Context, hash string) ([]TorrentFile, error) {
	query := url.Values{}
	query.Set("hash", hash)
	req, err := c.request(ctx, http.MethodGet, "/api/v2/torrents/files?"+query.Encode(), nil, "")
	if err != nil {
		return nil, err
	}
	var files []TorrentFile
	if err := c.doJSON(req, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *Client) AddMagnet(ctx context.Context, magnet string) error {
	form := url.Values{}
	form.Set("urls", magnet)
	if savePath := c.currentSavePath(); savePath != "" {
		form.Set("savepath", savePath)
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
	if savePath := c.currentSavePath(); savePath != "" {
		if err := writer.WriteField("savepath", savePath); err != nil {
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
	baseURL, username, password, loggedIn, blocked := c.currentLogin()
	if blocked != nil {
		return blocked
	}
	if baseURL == "" || username == "" || password == "" {
		return ErrNotConfigured
	}
	if loggedIn {
		return nil
	}

	form := url.Values{}
	form.Set("username", username)
	form.Set("password", password)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v2/auth/login", strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		c.block(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNoContent {
		c.markLoggedIn()
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "Ok." {
		err := fmt.Errorf("qbit login failed: status %d body %q", resp.StatusCode, string(body))
		c.block(err)
		return err
	}
	c.markLoggedIn()
	return nil
}

func (c *Client) request(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Request, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	baseURL := c.currentBaseURL()
	if baseURL == "" {
		return nil, ErrNotConfigured
	}
	req, err := http.NewRequestWithContext(ctx, method, baseURL+path, body)
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
		c.block(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		c.markLoggedOut()
		err := fmt.Errorf("qbit request forbidden")
		c.block(err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("qbit request failed: status %d body %q", resp.StatusCode, string(body))
		c.block(err)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) do(req *http.Request) error {
	resp, err := c.http.Do(req)
	if err != nil {
		c.block(err)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusForbidden {
		c.markLoggedOut()
		err := fmt.Errorf("qbit request forbidden")
		c.block(err)
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("qbit request failed: status %d body %q", resp.StatusCode, string(body))
		c.block(err)
		return err
	}
	return nil
}

func (c *Client) block(err error) {
	c.mu.Lock()
	c.blocked = err
	c.loggedIn = false
	c.mu.Unlock()
}

func (c *Client) markLoggedOut() {
	c.mu.Lock()
	c.loggedIn = false
	c.mu.Unlock()
}

func (c *Client) markLoggedIn() {
	c.mu.Lock()
	c.loggedIn = true
	c.mu.Unlock()
}

func (c *Client) currentLogin() (string, string, string, bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.baseURL, c.username, c.password, c.loggedIn, c.blocked
}

func (c *Client) currentBaseURL() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.baseURL
}

func (c *Client) currentSavePath() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.savePath
}
