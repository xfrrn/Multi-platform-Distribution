package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"multi-platform-distribution/internal/auth"
	"multi-platform-distribution/internal/config"
	"multi-platform-distribution/internal/domain"
	"multi-platform-distribution/internal/service"
	"multi-platform-distribution/internal/storage"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func TestAdminReleaseFlow(t *testing.T) {
	repo := newMemoryRepository(t)
	tempDir := t.TempDir()
	tokens := auth.NewTokenManager("test-secret", "test-issuer", time.Hour)
	services := Services{
		Apps:      service.NewAppService(repo),
		Releases:  service.NewReleaseService(repo),
		Artifacts: service.NewArtifactService(repo, storage.NewLocalStorage(tempDir, "http://updates.example.test")),
		Metadata:  service.NewMetadataService(repo),
		Auth:      service.NewAuthService(repo, tokens),
		Tokens:    tokens,
	}
	cfg := config.Config{
		APIKey:           "ci-key",
		StorageDriver:    "local",
		LocalStoragePath: tempDir,
	}
	router := NewRouter(cfg, services)
	server := httptest.NewServer(router)
	defer server.Close()

	assertStatus(t, request(t, server, http.MethodGet, "/api/apps", nil, ""), http.StatusUnauthorized)

	login := postJSON[loginResponse](t, server, "/api/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "secret-password",
	}, "")
	if login.AccessToken == "" {
		t.Fatal("expected access token")
	}
	bearer := "Bearer " + login.AccessToken

	app := postJSON[domain.App](t, server, "/api/apps", map[string]string{
		"name":            "Desktop App",
		"slug":            "desktop-app",
		"description":     "Updater fixture",
		"default_channel": "stable",
	}, bearer)
	if app.Slug != "desktop-app" {
		t.Fatalf("expected app slug desktop-app, got %s", app.Slug)
	}

	release := postJSON[domain.Release](t, server, "/api/apps/"+app.ID.String()+"/releases", map[string]any{
		"version":         "1.2.3",
		"channel":         "stable",
		"changelog":       "Ship the thing",
		"is_forced":       true,
		"staging_percent": 100,
		"published_at":    "2026-06-03T10:00:00Z",
	}, bearer)
	if release.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %s", release.Version)
	}

	artifact := uploadArtifact(t, server, release.ID, bearer)
	if artifact.Platform != "windows" || artifact.Arch != "x64" {
		t.Fatalf("unexpected artifact target %s/%s", artifact.Platform, artifact.Arch)
	}
	if artifact.FileSize != int64(len("installer payload")) {
		t.Fatalf("expected uploaded file size, got %d", artifact.FileSize)
	}
	if artifact.SHA512 == "" {
		t.Fatal("expected artifact sha512")
	}

	manifest := getJSON[domain.UpdateManifest](t, server, "/api/latest/desktop-app/update.json?platform=windows&arch=x64", "")
	if manifest.Version != "1.2.3" {
		t.Fatalf("expected manifest version 1.2.3, got %s", manifest.Version)
	}
	if len(manifest.Files) != 1 {
		t.Fatalf("expected one manifest file, got %d", len(manifest.Files))
	}
	if manifest.Files[0].URL != artifact.FileURL {
		t.Fatalf("expected manifest URL %s, got %s", artifact.FileURL, manifest.Files[0].URL)
	}

	yml := requestText(t, server, http.MethodGet, "/api/latest/desktop-app/latest.yml?platform=windows&arch=x64", nil, "")
	if !strings.Contains(yml, "version: 1.2.3") || !strings.Contains(yml, "sha512:") {
		t.Fatalf("latest.yml missing expected fields:\n%s", yml)
	}

	appcast := requestText(t, server, http.MethodGet, "/api/latest/desktop-app/appcast.xml?platform=windows&arch=x64", nil, "")
	if !strings.Contains(appcast, `<rss version="2.0"`) || !strings.Contains(appcast, `sparkle:version="1.2.3"`) {
		t.Fatalf("appcast.xml missing expected fields:\n%s", appcast)
	}

	redirect := requestNoRedirect(t, server, http.MethodGet, "/api/artifacts/"+artifact.ID.String()+"/download", nil, "")
	defer redirect.Body.Close()
	if redirect.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected download endpoint to return 307, got %d", redirect.StatusCode)
	}
	location := redirect.Header.Get("Location")
	if location != artifact.FileURL {
		t.Fatalf("expected redirect location %s, got %s", artifact.FileURL, location)
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect location: %v", err)
	}
	fileResp := request(t, server, http.MethodGet, parsed.Path, nil, "")
	defer fileResp.Body.Close()
	if fileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected static file status 200, got %d", fileResp.StatusCode)
	}
	body, err := io.ReadAll(fileResp.Body)
	if err != nil {
		t.Fatalf("read download body: %v", err)
	}
	if string(body) != "installer payload" {
		t.Fatalf("expected downloaded artifact content, got %q", string(body))
	}
}

func TestAdminAPIKeyCanAccessManagementRoutes(t *testing.T) {
	repo := newMemoryRepository(t)
	tokens := auth.NewTokenManager("test-secret", "test-issuer", time.Hour)
	router := NewRouter(config.Config{APIKey: "ci-key"}, Services{
		Apps:      service.NewAppService(repo),
		Releases:  service.NewReleaseService(repo),
		Artifacts: service.NewArtifactService(repo, storage.NewLocalStorage(t.TempDir(), "")),
		Metadata:  service.NewMetadataService(repo),
		Auth:      service.NewAuthService(repo, tokens),
		Tokens:    tokens,
	})
	server := httptest.NewServer(router)
	defer server.Close()

	reqBody := strings.NewReader(`{"name":"CI App","slug":"ci-app"}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/apps", reqBody)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "ci-key")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected API key to create app, got status %d", resp.StatusCode)
	}
}

type loginResponse struct {
	AccessToken string           `json:"access_token"`
	TokenType   string           `json:"token_type"`
	ExpiresAt   time.Time        `json:"expires_at"`
	Admin       domain.AdminUser `json:"admin"`
}

func postJSON[T any](t *testing.T, server *httptest.Server, path string, payload any, bearer string) T {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	resp := request(t, server, http.MethodPost, path, bytes.NewReader(body), bearer)
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST %s got status %d: %s", path, resp.StatusCode, string(data))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func getJSON[T any](t *testing.T, server *httptest.Server, path, bearer string) T {
	t.Helper()

	resp := request(t, server, http.MethodGet, path, nil, bearer)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("GET %s got status %d: %s", path, resp.StatusCode, string(data))
	}

	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func uploadArtifact(t *testing.T, server *httptest.Server, releaseID uuid.UUID, bearer string) domain.Artifact {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"platform":  "windows",
		"arch":      "x64",
		"file_type": "exe",
	} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("write field: %v", err)
		}
	}
	part, err := writer.CreateFormFile("file", "desktop-app.exe")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("installer payload")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPost, server.URL+"/api/releases/"+releaseID.String()+"/artifacts", &body)
	if err != nil {
		t.Fatalf("create upload request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", bearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("upload artifact: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("upload got status %d: %s", resp.StatusCode, string(data))
	}

	var artifact domain.Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		t.Fatalf("decode artifact: %v", err)
	}
	return artifact
}

func requestText(t *testing.T, server *httptest.Server, method, path string, body io.Reader, bearer string) string {
	t.Helper()

	resp := request(t, server, method, path, body, bearer)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("%s %s got status %d: %s", method, path, resp.StatusCode, string(data))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	return string(data)
}

func request(t *testing.T, server *httptest.Server, method, path string, body io.Reader, bearer string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return resp
}

func requestNoRedirect(t *testing.T, server *httptest.Server, method, path string, body io.Reader, bearer string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, server.URL+path, body)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if bearer != "" {
		req.Header.Set("Authorization", bearer)
	}

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	return resp
}

func assertStatus(t *testing.T, resp *http.Response, status int) {
	t.Helper()
	defer resp.Body.Close()
	if resp.StatusCode != status {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("expected status %d, got %d: %s", status, resp.StatusCode, string(data))
	}
}

type memoryRepository struct {
	mu        sync.Mutex
	apps      map[uuid.UUID]domain.App
	appSlugs  map[string]uuid.UUID
	releases  map[uuid.UUID]domain.Release
	artifacts map[uuid.UUID]domain.Artifact
	admins    map[string]domain.AdminUser
}

func newMemoryRepository(t *testing.T) *memoryRepository {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("secret-password"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	admin := domain.AdminUser{
		ID:           uuid.New(),
		Email:        "admin@example.com",
		Name:         "Admin",
		PasswordHash: string(passwordHash),
		IsActive:     true,
		CreatedAt:    time.Now().UTC(),
	}

	return &memoryRepository{
		apps:      make(map[uuid.UUID]domain.App),
		appSlugs:  make(map[string]uuid.UUID),
		releases:  make(map[uuid.UUID]domain.Release),
		artifacts: make(map[uuid.UUID]domain.Artifact),
		admins: map[string]domain.AdminUser{
			admin.Email: admin,
		},
	}
}

func (r *memoryRepository) CreateApp(ctx context.Context, app *domain.App) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.appSlugs[app.Slug]; exists {
		return errors.New("app slug already exists")
	}
	app.CreatedAt = time.Now().UTC()
	r.apps[app.ID] = *app
	r.appSlugs[app.Slug] = app.ID
	return nil
}

func (r *memoryRepository) ListApps(ctx context.Context) ([]domain.App, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	apps := make([]domain.App, 0, len(r.apps))
	for _, app := range r.apps {
		apps = append(apps, app)
	}
	sort.Slice(apps, func(i, j int) bool {
		return apps[i].CreatedAt.After(apps[j].CreatedAt)
	})
	return apps, nil
}

func (r *memoryRepository) GetAppByID(ctx context.Context, id uuid.UUID) (domain.App, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.getAppByID(ctx, id)
}

func (r *memoryRepository) GetAppBySlug(ctx context.Context, slug string) (domain.App, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.App{}, err
	}
	id, ok := r.appSlugs[slug]
	if !ok {
		return domain.App{}, domain.ErrNotFound
	}
	return r.apps[id], nil
}

func (r *memoryRepository) CreateRelease(ctx context.Context, release *domain.Release) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := r.getAppByID(ctx, release.AppID); err != nil {
		return err
	}
	release.CreatedAt = time.Now().UTC()
	r.releases[release.ID] = *release
	return nil
}

func (r *memoryRepository) GetReleaseByID(ctx context.Context, id uuid.UUID) (domain.Release, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.Release{}, err
	}
	release, ok := r.releases[id]
	if !ok {
		return domain.Release{}, domain.ErrNotFound
	}
	return release, nil
}

func (r *memoryRepository) ListReleasesByApp(ctx context.Context, appID uuid.UUID) ([]domain.Release, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	releases := make([]domain.Release, 0)
	for _, release := range r.releases {
		if release.AppID == appID {
			releases = append(releases, release)
		}
	}
	sort.Slice(releases, func(i, j int) bool {
		return releases[i].PublishedAt.After(releases[j].PublishedAt)
	})
	return releases, nil
}

func (r *memoryRepository) GetLatestRelease(ctx context.Context, appID uuid.UUID, channel string) (domain.Release, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.Release{}, err
	}
	var latest domain.Release
	found := false
	now := time.Now().UTC()
	for _, release := range r.releases {
		if release.AppID != appID || release.Channel != channel || release.PublishedAt.After(now) {
			continue
		}
		if !found ||
			release.PublishedAt.After(latest.PublishedAt) ||
			(release.PublishedAt.Equal(latest.PublishedAt) && release.CreatedAt.After(latest.CreatedAt)) {
			latest = release
			found = true
		}
	}
	if !found {
		return domain.Release{}, domain.ErrNotFound
	}
	return latest, nil
}

func (r *memoryRepository) CreateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	if _, ok := r.releases[artifact.ReleaseID]; !ok {
		return domain.ErrNotFound
	}
	artifact.CreatedAt = time.Now().UTC()
	r.artifacts[artifact.ID] = *artifact
	return nil
}

func (r *memoryRepository) GetArtifactByID(ctx context.Context, id uuid.UUID) (domain.Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.Artifact{}, err
	}
	artifact, ok := r.artifacts[id]
	if !ok {
		return domain.Artifact{}, domain.ErrNotFound
	}
	return artifact, nil
}

func (r *memoryRepository) ListArtifactsByRelease(ctx context.Context, releaseID uuid.UUID) ([]domain.Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	artifacts := make([]domain.Artifact, 0)
	for _, artifact := range r.artifacts {
		if artifact.ReleaseID == releaseID {
			artifacts = append(artifacts, artifact)
		}
	}
	sort.Slice(artifacts, func(i, j int) bool {
		return fmt.Sprintf("%s/%s/%s", artifacts[i].Platform, artifacts[i].Arch, artifacts[i].FileType) <
			fmt.Sprintf("%s/%s/%s", artifacts[j].Platform, artifacts[j].Arch, artifacts[j].FileType)
	})
	return artifacts, nil
}

func (r *memoryRepository) CreateAdminUser(ctx context.Context, user *domain.AdminUser) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	email := strings.ToLower(user.Email)
	if _, exists := r.admins[email]; exists {
		return errors.New("admin email already exists")
	}
	user.Email = email
	user.CreatedAt = time.Now().UTC()
	r.admins[email] = *user
	return nil
}

func (r *memoryRepository) GetAdminUserByEmail(ctx context.Context, email string) (domain.AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.AdminUser{}, err
	}
	admin, ok := r.admins[strings.ToLower(strings.TrimSpace(email))]
	if !ok {
		return domain.AdminUser{}, domain.ErrNotFound
	}
	return admin, nil
}

func (r *memoryRepository) CountAdminUsers(ctx context.Context) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return len(r.admins), nil
}

func (r *memoryRepository) getAppByID(ctx context.Context, id uuid.UUID) (domain.App, error) {
	if err := ctx.Err(); err != nil {
		return domain.App{}, err
	}
	app, ok := r.apps[id]
	if !ok {
		return domain.App{}, domain.ErrNotFound
	}
	return app, nil
}
