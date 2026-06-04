package anyshare

import (
	"bytes"
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Config struct {
	BaseURL         string
	SharingLink     string
	UploadPath      string
	Cookie          string
	Timeout         time.Duration
	RefreshInterval time.Duration
}

type Client struct {
	baseURL         string
	sharingLink     string
	sharingID       string
	uploadPath      string
	loginCookie     string
	refreshInterval time.Duration
	authorization   string
	httpClient      *http.Client
	stopRefresh     context.CancelFunc
	mu              sync.Mutex
}

type UploadResult struct {
	DocID  string
	Rev    string
	Name   string
	Size   int64
	SHA512 string
}

type DownloadResult struct {
	URL  string
	Name string
	Size int64
}

type uploadConfig struct {
	URL    string
	Fields map[string]string
	DocID  string
	Rev    string
}

var sharingIDPattern = regexp.MustCompile(`^[A-Za-z0-9]{16,64}$`)

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	baseURL := strings.TrimRight(cfg.BaseURL, "/")
	if baseURL == "" {
		return nil, errors.New("ANYSHARE_BASE_URL is required")
	}
	sharingLink, sharingID, err := parseSharingLink(baseURL, cfg.SharingLink)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.UploadPath) == "" {
		return nil, errors.New("ANYSHARE_UPLOAD_PATH is required")
	}
	if !strings.HasPrefix(cfg.UploadPath, "gns://") {
		return nil, errors.New("ANYSHARE_UPLOAD_PATH must be a gns:// path")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	refreshInterval := cfg.RefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = 30 * time.Minute
	}
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("create anyshare cookie jar: %w", err)
	}
	client := &Client{
		baseURL:         baseURL,
		sharingLink:     sharingLink,
		sharingID:       sharingID,
		uploadPath:      strings.TrimSpace(cfg.UploadPath),
		loginCookie:     strings.TrimSpace(cfg.Cookie),
		refreshInterval: refreshInterval,
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
	if client.loginCookie != "" {
		if err := client.setCookie(client.loginCookie); err != nil {
			return nil, err
		}
		client.mu.Lock()
		err := client.refreshAuthorizationLocked(ctx, false)
		client.mu.Unlock()
		if err != nil {
			return nil, err
		}
		client.startAuthorizationRefresher()
	}
	return client, nil
}

func (c *Client) Close() {
	if c.stopRefresh != nil {
		c.stopRefresh()
	}
}

func (c *Client) Upload(ctx context.Context, fileName string, body io.Reader) (UploadResult, error) {
	if err := c.ensureAuthorization(ctx); err != nil {
		return UploadResult{}, err
	}
	if body == nil {
		return UploadResult{}, errors.New("file body is required")
	}
	name := sanitizeName(fileName)
	if name == "" {
		return UploadResult{}, errors.New("file name is required")
	}
	temp, err := os.CreateTemp("", "mpd-anyshare-upload-*")
	if err != nil {
		return UploadResult{}, err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	hash := sha512.New()
	size, err := io.Copy(io.MultiWriter(temp, hash), body)
	if err != nil {
		return UploadResult{}, err
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return UploadResult{}, err
	}

	cfg, err := c.beginUpload(ctx, name, size)
	if err != nil {
		return UploadResult{}, err
	}
	if err := c.doUpload(ctx, cfg, name, temp); err != nil {
		return UploadResult{}, err
	}
	if err := c.endUpload(ctx, cfg); err != nil {
		return UploadResult{}, err
	}

	return UploadResult{
		DocID:  cfg.DocID,
		Rev:    cfg.Rev,
		Name:   name,
		Size:   size,
		SHA512: base64.StdEncoding.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (c *Client) ResolveDownload(ctx context.Context, docID, rev, name string) (DownloadResult, error) {
	if err := c.ensureAuthorization(ctx); err != nil {
		return DownloadResult{}, err
	}
	if strings.TrimSpace(docID) == "" {
		return DownloadResult{}, errors.New("anyshare docid is required")
	}
	saveName := sanitizeName(name)
	if saveName == "" {
		saveName = "download"
	}
	var response struct {
		AuthRequest []string `json:"authrequest"`
		Size        int64    `json:"size"`
	}
	if err := c.postJSON(ctx, "/api/efast/v1/file/osdownload", map[string]any{
		"docid":     docID,
		"authtype":  "QUERY_STRING",
		"savename":  saveName,
		"use_https": true,
		"rev":       strings.TrimSpace(rev),
	}, &response); err != nil {
		return DownloadResult{}, err
	}
	downloadURL, err := parseDownloadAuth(response.AuthRequest)
	if err != nil {
		return DownloadResult{}, err
	}
	return DownloadResult{URL: downloadURL, Name: saveName, Size: response.Size}, nil
}

func parseSharingLink(baseURL, sharingLink string) (string, string, error) {
	sharingLink = strings.TrimSpace(sharingLink)
	if sharingLink == "" {
		return "", "", errors.New("ANYSHARE_SHARING_LINK is required")
	}
	share, err := url.Parse(sharingLink)
	if err != nil {
		return "", "", fmt.Errorf("parse ANYSHARE_SHARING_LINK: %w", err)
	}
	base, err := url.Parse(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return "", "", fmt.Errorf("parse ANYSHARE_BASE_URL: %w", err)
	}
	if share.Scheme != base.Scheme || share.Host != base.Host {
		return "", "", errors.New("ANYSHARE_SHARING_LINK must use ANYSHARE_BASE_URL host")
	}
	parts := strings.Split(strings.Trim(share.Path, "/"), "/")
	linkIndex := -1
	for i, part := range parts {
		if part == "link" {
			linkIndex = i
		}
	}
	if linkIndex < 0 || linkIndex == len(parts)-1 {
		return "", "", errors.New("ANYSHARE_SHARING_LINK must contain /link/{sharing_id}")
	}
	sharingID := parts[linkIndex+1]
	if !sharingIDPattern.MatchString(sharingID) {
		return "", "", errors.New("invalid anyshare sharing id")
	}
	return strings.TrimRight(baseURL, "/") + "/link/" + sharingID, sharingID, nil
}

func (c *Client) visitSharingLink(ctx context.Context) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("parse ANYSHARE_BASE_URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.sharingLink, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("visit anyshare sharing link: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return responseError("visit anyshare sharing link", resp)
	}

	tokenName := "link_token:" + c.sharingID
	for _, header := range resp.Header.Values("Set-Cookie") {
		if value, ok := parseRawCookie(header, tokenName); ok {
			c.authorization = "Bearer " + value
			return nil
		}
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == tokenName {
			c.authorization = "Bearer " + cookie.Value
			return nil
		}
	}
	for _, cookie := range c.httpClient.Jar.Cookies(base) {
		if cookie.Name == tokenName {
			c.authorization = "Bearer " + cookie.Value
			return nil
		}
	}
	return errors.New("failed to get anyshare anonymous token")
}

func (c *Client) ensureAuthorization(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.loginCookie != "" {
		if c.authorization != "" {
			return nil
		}
		return c.refreshAuthorizationLocked(ctx, false)
	}
	if c.authorization != "" {
		return nil
	}
	return c.visitSharingLink(ctx)
}

func (c *Client) startAuthorizationRefresher() {
	ctx, cancel := context.WithCancel(context.Background())
	c.stopRefresh = cancel
	go func() {
		ticker := time.NewTicker(c.refreshInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				c.mu.Lock()
				err := c.refreshAuthorizationLocked(ctx, false)
				c.mu.Unlock()
				if err != nil {
					log.Printf("refresh anyshare authorization: %v", err)
				}
			}
		}
	}()
}

func (c *Client) setCookie(rawCookie string) error {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return fmt.Errorf("parse ANYSHARE_BASE_URL: %w", err)
	}
	cookies := make([]*http.Cookie, 0)
	for _, item := range strings.Split(rawCookie, ";") {
		name, value, ok := strings.Cut(strings.TrimSpace(item), "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		value = strings.TrimSpace(value)
		cookies = append(cookies, &http.Cookie{Name: name, Value: value, Path: "/"})
		if name == "Authorization" {
			c.authorization = normalizeAuthorizationCookie(value)
		}
	}
	if len(cookies) == 0 {
		return errors.New("ANYSHARE_COOKIE did not contain any cookies")
	}
	c.httpClient.Jar.SetCookies(base, cookies)
	return nil
}

func (c *Client) refreshAuthorizationLocked(ctx context.Context, force bool) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/anyshare/oauth2/login/refreshToken", nil)
	if err != nil {
		return err
	}
	query := req.URL.Query()
	if force {
		query.Set("force", "true")
	} else {
		query.Set("force", "false")
	}
	req.URL.RawQuery = query.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("refresh anyshare authorization: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError("refresh anyshare authorization", resp)
	}
	if c.updateAuthorizationFromCookies(resp) {
		return nil
	}
	return errors.New("anyshare Authorization cookie is missing")
}

func (c *Client) updateAuthorizationFromCookies(resp *http.Response) bool {
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "Authorization" && cookie.Value != "" {
			c.authorization = normalizeAuthorizationCookie(cookie.Value)
			return true
		}
	}

	base, err := url.Parse(c.baseURL)
	if err == nil {
		for _, cookie := range c.httpClient.Jar.Cookies(base) {
			if cookie.Name == "Authorization" && cookie.Value != "" {
				c.authorization = normalizeAuthorizationCookie(cookie.Value)
				return true
			}
		}
	}

	for _, header := range resp.Header.Values("Set-Cookie") {
		if value, ok := parseRawCookie(header, "Authorization"); ok && value != "" {
			c.authorization = normalizeAuthorizationCookie(value)
			return true
		}
	}
	return false
}

func (c *Client) beginUpload(ctx context.Context, name string, size int64) (uploadConfig, error) {
	var response struct {
		AuthRequest []string `json:"authrequest"`
		DocID       string   `json:"docid"`
		Rev         string   `json:"rev"`
	}
	err := c.postJSON(ctx, "/api/efast/v1/file/osbeginupload", map[string]any{
		"use_https":    true,
		"reqmethod":    "POST",
		"name":         name,
		"docid":        c.uploadPath,
		"ondup":        3,
		"length":       size,
		"client_mtime": time.Now().UnixMilli(),
		"gns_dir_path": c.uploadPath,
	}, &response)
	if err != nil {
		return uploadConfig{}, err
	}
	uploadURL, fields, err := parseUploadAuth(response.AuthRequest)
	if err != nil {
		return uploadConfig{}, err
	}
	if response.DocID == "" || response.Rev == "" {
		return uploadConfig{}, errors.New("anyshare beginupload response missing docid or rev")
	}
	return uploadConfig{URL: uploadURL, Fields: fields, DocID: response.DocID, Rev: response.Rev}, nil
}

func (c *Client) doUpload(ctx context.Context, cfg uploadConfig, name string, file io.Reader) error {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range cfg.Fields {
		if err := writer.WriteField(key, value); err != nil {
			return err
		}
	}
	part, err := writer.CreateFormFile("file", name)
	if err != nil {
		return err
	}
	if _, err := io.Copy(part, file); err != nil {
		return err
	}
	if err := writer.Close(); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.URL, &body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload anyshare object: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return responseError("upload anyshare object", resp)
	}
	return nil
}

func (c *Client) endUpload(ctx context.Context, cfg uploadConfig) error {
	return c.postJSON(ctx, "/api/efast/v1/file/osendupload", map[string]any{
		"docid":    cfg.DocID,
		"rev":      cfg.Rev,
		"csflevel": 0,
	}, nil)
}

func (c *Client) postJSON(ctx context.Context, endpoint string, payload any, out any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	for attempt := 0; attempt < 2; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewReader(data))
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")

		c.mu.Lock()
		authorization := c.authorization
		c.mu.Unlock()
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("call anyshare %s: %w", endpoint, err)
		}
		if resp.StatusCode == http.StatusUnauthorized && c.loginCookie != "" && attempt == 0 {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
			resp.Body.Close()

			c.mu.Lock()
			c.authorization = ""
			err = c.refreshAuthorizationLocked(ctx, true)
			c.mu.Unlock()
			if err != nil {
				return err
			}
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return responseError("call anyshare "+endpoint, resp)
		}
		if out == nil {
			return nil
		}
		return json.NewDecoder(resp.Body).Decode(out)
	}
	return errors.New("call anyshare failed after authorization refresh")
}

func parseDownloadAuth(auth []string) (string, error) {
	if len(auth) != 2 {
		return "", errors.New("unexpected anyshare download authrequest length")
	}
	if auth[0] != http.MethodGet {
		return "", errors.New("unexpected anyshare download method")
	}
	if strings.TrimSpace(auth[1]) == "" {
		return "", errors.New("empty anyshare download url")
	}
	return auth[1], nil
}

func parseUploadAuth(auth []string) (string, map[string]string, error) {
	if len(auth) != 7 {
		return "", nil, errors.New("unexpected anyshare upload authrequest length")
	}
	if auth[0] != http.MethodPost {
		return "", nil, errors.New("unexpected anyshare upload method")
	}
	if strings.TrimSpace(auth[1]) == "" {
		return "", nil, errors.New("empty anyshare upload url")
	}
	fields := make(map[string]string, 5)
	for _, item := range auth[2:] {
		key, value, ok := strings.Cut(item, ": ")
		if !ok || key == "" {
			return "", nil, errors.New("unexpected anyshare upload auth field")
		}
		fields[key] = value
	}
	for _, key := range []string{"AWSAccessKeyId", "Content-Type", "Policy", "Signature", "key"} {
		if _, ok := fields[key]; !ok {
			return "", nil, fmt.Errorf("anyshare upload auth missing %s", key)
		}
	}
	return auth[1], fields, nil
}

func parseRawCookie(header, name string) (string, bool) {
	prefix := name + "="
	for _, part := range strings.Split(header, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, prefix) {
			return strings.TrimPrefix(part, prefix), true
		}
	}
	return "", false
}

func normalizeAuthorizationCookie(value string) string {
	value = strings.TrimSpace(value)
	if decoded, err := url.QueryUnescape(value); err == nil && strings.HasPrefix(decoded, "Bearer ") {
		return decoded
	}
	return value
}

func sanitizeName(name string) string {
	replacer := strings.NewReplacer("\\", "_", "/", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")
	return strings.TrimSpace(replacer.Replace(path.Base(strings.TrimSpace(name))))
}

func responseError(action string, resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	message := strings.TrimSpace(string(data))
	if message == "" {
		message = resp.Status
	}
	return fmt.Errorf("%s: status %d: %s", action, resp.StatusCode, message)
}
