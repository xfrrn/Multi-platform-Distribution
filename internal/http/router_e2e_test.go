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

	"multi-platform-distribution/internal/anyshare"
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
		Stats:     service.NewStatsService(repo),
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
	if artifact.FileName != "desktop-app.exe" {
		t.Fatalf("expected artifact file name desktop-app.exe, got %s", artifact.FileName)
	}
	if artifact.FileSize != int64(len("installer payload")) {
		t.Fatalf("expected uploaded file size, got %d", artifact.FileSize)
	}
	if artifact.SHA512 == "" {
		t.Fatal("expected artifact sha512")
	}
	artifact = replaceArtifactFile(t, server, artifact.ID, bearer)
	if artifact.FileName != "desktop-app-v2.exe" {
		t.Fatalf("expected replaced file name, got %s", artifact.FileName)
	}
	if artifact.FileSize != int64(len("replacement payload")) {
		t.Fatalf("expected replaced file size, got %d", artifact.FileSize)
	}

	manifest := getJSON[domain.UpdateManifest](t, server, "/api/latest/desktop-app/update.json?platform=windows&arch=x64&client_id=client-a", "")
	if manifest.Version != "1.2.3" {
		t.Fatalf("expected manifest version 1.2.3, got %s", manifest.Version)
	}
	if len(manifest.Files) != 1 {
		t.Fatalf("expected one manifest file, got %d", len(manifest.Files))
	}
	if manifest.Files[0].URL != artifact.FileURL {
		t.Fatalf("expected manifest URL %s, got %s", artifact.FileURL, manifest.Files[0].URL)
	}

	artifactList := getJSON[listResponse[domain.Artifact]](t, server, "/api/releases/"+release.ID.String()+"/artifacts", bearer)
	if len(artifactList.Items) != 1 {
		t.Fatalf("expected one artifact in list, got %d", len(artifactList.Items))
	}

	yml := requestText(t, server, http.MethodGet, "/api/latest/desktop-app/latest.yml?platform=windows&arch=x64", nil, "")
	if !strings.Contains(yml, "version: 1.2.3") || !strings.Contains(yml, "sha512:") {
		t.Fatalf("latest.yml missing expected fields:\n%s", yml)
	}

	appcast := requestText(t, server, http.MethodGet, "/api/latest/desktop-app/appcast.xml?platform=windows&arch=x64", nil, "")
	if !strings.Contains(appcast, `<rss version="2.0"`) || !strings.Contains(appcast, `sparkle:version="1.2.3"`) {
		t.Fatalf("appcast.xml missing expected fields:\n%s", appcast)
	}

	redirect := requestNoRedirect(t, server, http.MethodGet, "/api/artifacts/"+artifact.ID.String()+"/download?client_id=client-a", nil, "")
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
	if string(body) != "replacement payload" {
		t.Fatalf("expected downloaded artifact content, got %q", string(body))
	}

	summary := getJSON[domain.StatsSummary](t, server, "/api/stats/summary", bearer)
	if summary.TotalUpdateRequests == 0 {
		t.Fatal("expected update request stats")
	}
	if summary.TotalDownloads == 0 {
		t.Fatal("expected download stats")
	}
	if summary.ForcedDownloads7d == 0 {
		t.Fatal("expected forced download stats")
	}
	if len(summary.PlatformBreakdown) == 0 || summary.PlatformBreakdown[0].Key != "windows" {
		t.Fatalf("expected windows platform breakdown, got %#v", summary.PlatformBreakdown)
	}
	if len(summary.VersionBreakdown) == 0 || summary.VersionBreakdown[0].Key != "1.2.3" {
		t.Fatalf("expected version breakdown, got %#v", summary.VersionBreakdown)
	}
	releaseSummary := getJSON[domain.StatsSummary](t, server, "/api/releases/"+release.ID.String()+"/stats", bearer)
	if releaseSummary.TotalDownloads == 0 {
		t.Fatal("expected release download stats")
	}
	updateEvents := getJSON[listResponse[domain.UpdateRequestEvent]](t, server, "/api/apps/"+app.ID.String()+"/stats/update-requests?client_id=client-a&limit=5&offset=0", bearer)
	if len(updateEvents.Items) == 0 || !updateEvents.Items[0].Matched || updateEvents.Items[0].ClientID != "client-a" {
		t.Fatalf("expected matched update event, got %#v", updateEvents.Items)
	}
	downloadEvents := getJSON[listResponse[domain.DownloadEvent]](t, server, "/api/apps/"+app.ID.String()+"/stats/downloads?client_id=client-a", bearer)
	if len(downloadEvents.Items) == 0 || downloadEvents.Items[0].ArtifactID != artifact.ID || downloadEvents.Items[0].ClientID != "client-a" {
		t.Fatalf("expected download event for artifact %s, got %#v", artifact.ID, downloadEvents.Items)
	}

	assertStatus(t, request(t, server, http.MethodDelete, "/api/artifacts/"+artifact.ID.String(), nil, bearer), http.StatusNoContent)
	reuploaded := uploadArtifact(t, server, release.ID, bearer)
	if reuploaded.ID == artifact.ID {
		t.Fatal("expected upload after archive to create a new artifact")
	}
	artifactList = getJSON[listResponse[domain.Artifact]](t, server, "/api/releases/"+release.ID.String()+"/artifacts", bearer)
	if len(artifactList.Items) != 1 || artifactList.Items[0].ID != reuploaded.ID {
		t.Fatalf("expected one active reuploaded artifact, got %#v", artifactList.Items)
	}
}

func TestAnyshareArtifactFlow(t *testing.T) {
	repo := newMemoryRepository(t)
	anyshareServer, anyshareClient := newMockAnyshare(t)
	defer anyshareServer.Close()

	tempDir := t.TempDir()
	tokens := auth.NewTokenManager("test-secret", "test-issuer", time.Hour)
	cfg := config.Config{
		APIKey:           "ci-key",
		StorageDriver:    "local",
		LocalStoragePath: tempDir,
		PublicBaseURL:    "http://updates.example.test",
		AnyshareEnabled:  true,
	}
	router := NewRouter(cfg, Services{
		Apps:      service.NewAppService(repo),
		Releases:  service.NewReleaseService(repo),
		Artifacts: service.NewArtifactServiceWithAnyshare(repo, storage.NewLocalStorage(tempDir, cfg.PublicBaseURL), anyshareClient, cfg.PublicBaseURL),
		Metadata:  service.NewMetadataService(repo),
		Stats:     service.NewStatsService(repo),
		Auth:      service.NewAuthService(repo, tokens),
		Tokens:    tokens,
	})
	server := httptest.NewServer(router)
	defer server.Close()

	login := postJSON[loginResponse](t, server, "/api/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "secret-password",
	}, "")
	bearer := "Bearer " + login.AccessToken

	app := postJSON[domain.App](t, server, "/api/apps", map[string]string{
		"name": "Anyshare App",
		"slug": "anyshare-app",
	}, bearer)
	release := postJSON[domain.Release](t, server, "/api/apps/"+app.ID.String()+"/releases", map[string]any{
		"version":         "3.0.0",
		"channel":         "stable",
		"staging_percent": 100,
		"published_at":    "2026-06-03T10:00:00Z",
	}, bearer)

	artifact := uploadArtifactWithSource(t, server, release.ID, bearer, "anyshare")
	if artifact.SourceType != "anyshare" {
		t.Fatalf("expected anyshare artifact, got %s", artifact.SourceType)
	}
	if artifact.AnyshareDocID == "" || artifact.AnyshareRev == "" || artifact.AnyshareName == "" {
		t.Fatalf("expected anyshare metadata, got %#v", artifact)
	}
	expectedStableURL := "http://updates.example.test/api/artifacts/" + artifact.ID.String() + "/download/desktop-app.exe"
	if artifact.FileURL != expectedStableURL {
		t.Fatalf("expected stable artifact url %s, got %s", expectedStableURL, artifact.FileURL)
	}

	replaced := uploadArtifactWithSource(t, server, release.ID, bearer, "anyshare")
	if replaced.ID != artifact.ID {
		t.Fatalf("expected duplicate anyshare upload to replace artifact %s, got %s", artifact.ID, replaced.ID)
	}
	artifacts := getJSON[listResponse[domain.Artifact]](t, server, "/api/releases/"+release.ID.String()+"/artifacts", bearer)
	if len(artifacts.Items) != 1 {
		t.Fatalf("expected duplicate upload to keep one artifact, got %d", len(artifacts.Items))
	}

	manifest := getJSON[domain.UpdateManifest](t, server, "/api/latest/anyshare-app/update.json?platform=windows&arch=x64", "")
	if len(manifest.Files) != 1 || manifest.Files[0].URL != expectedStableURL {
		t.Fatalf("expected manifest to use stable url %s, got %#v", expectedStableURL, manifest.Files)
	}

	redirect := requestNoRedirect(t, server, http.MethodGet, "/api/artifacts/"+artifact.ID.String()+"/download?client_id=client-a", nil, "")
	defer redirect.Body.Close()
	if redirect.StatusCode != http.StatusTemporaryRedirect {
		t.Fatalf("expected anyshare download redirect, got %d", redirect.StatusCode)
	}
	location := redirect.Header.Get("Location")
	if !strings.Contains(location, "/direct/desktop-app.exe?token=download") {
		t.Fatalf("expected resolved anyshare direct url, got %s", location)
	}
}

func TestArchiveAndStagingFlow(t *testing.T) {
	repo := newMemoryRepository(t)
	server, bearer := newTestServer(t, repo)
	defer server.Close()

	app := postJSON[domain.App](t, server, "/api/apps", map[string]string{
		"name":            "Staged App",
		"slug":            "staged-app",
		"default_channel": "stable",
	}, bearer)

	stable := postJSON[domain.Release](t, server, "/api/apps/"+app.ID.String()+"/releases", map[string]any{
		"version":         "1.0.0",
		"channel":         "stable",
		"staging_percent": 100,
		"published_at":    "2026-06-01T10:00:00Z",
	}, bearer)
	_ = uploadArtifact(t, server, stable.ID, bearer)

	staged := postJSON[domain.Release](t, server, "/api/apps/"+app.ID.String()+"/releases", map[string]any{
		"version":         "2.0.0",
		"channel":         "stable",
		"staging_percent": 1,
		"published_at":    "2026-06-02T10:00:00Z",
	}, bearer)
	artifact := uploadArtifact(t, server, staged.ID, bearer)

	noClientManifest := getJSON[domain.UpdateManifest](t, server, "/api/latest/staged-app/update.json", "")
	if noClientManifest.Version != "1.0.0" {
		t.Fatalf("expected no-client latest to avoid staged release, got %s", noClientManifest.Version)
	}

	hitClient := findStagingClient(t, server, "staged-app", "2.0.0")
	first := getJSON[domain.UpdateManifest](t, server, "/api/latest/staged-app/update.json?client_id="+hitClient, "")
	second := getJSON[domain.UpdateManifest](t, server, "/api/latest/staged-app/update.json?client_id="+hitClient, "")
	if first.Version != "2.0.0" || second.Version != "2.0.0" {
		t.Fatalf("expected stable staging hit, got %s then %s", first.Version, second.Version)
	}

	assertStatus(t, request(t, server, http.MethodDelete, "/api/artifacts/"+artifact.ID.String(), nil, bearer), http.StatusNoContent)
	afterArtifactArchive := getJSON[domain.UpdateManifest](t, server, "/api/latest/staged-app/update.json?client_id="+hitClient, "")
	if afterArtifactArchive.Version != "2.0.0" || len(afterArtifactArchive.Files) != 0 {
		t.Fatalf("expected archived artifact hidden from manifest, got version %s files %d", afterArtifactArchive.Version, len(afterArtifactArchive.Files))
	}

	assertStatus(t, request(t, server, http.MethodDelete, "/api/releases/"+staged.ID.String(), nil, bearer), http.StatusNoContent)
	afterReleaseArchive := getJSON[domain.UpdateManifest](t, server, "/api/latest/staged-app/update.json?client_id="+hitClient, "")
	if afterReleaseArchive.Version != "1.0.0" {
		t.Fatalf("expected latest to fall back after release archive, got %s", afterReleaseArchive.Version)
	}

	assertStatus(t, request(t, server, http.MethodDelete, "/api/apps/"+app.ID.String(), nil, bearer), http.StatusNoContent)
	assertStatus(t, request(t, server, http.MethodGet, "/api/latest/staged-app/update.json", nil, ""), http.StatusNotFound)
}

func TestAdminAPIKeyCanAccessManagementRoutes(t *testing.T) {
	repo := newMemoryRepository(t)
	tokens := auth.NewTokenManager("test-secret", "test-issuer", time.Hour)
	router := NewRouter(config.Config{APIKey: "ci-key"}, Services{
		Apps:      service.NewAppService(repo),
		Releases:  service.NewReleaseService(repo),
		Artifacts: service.NewArtifactService(repo, storage.NewLocalStorage(t.TempDir(), "")),
		Metadata:  service.NewMetadataService(repo),
		Stats:     service.NewStatsService(repo),
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

type listResponse[T any] struct {
	Items []T `json:"items"`
}

func newTestServer(t *testing.T, repo *memoryRepository) (*httptest.Server, string) {
	t.Helper()

	tempDir := t.TempDir()
	tokens := auth.NewTokenManager("test-secret", "test-issuer", time.Hour)
	router := NewRouter(config.Config{
		APIKey:           "ci-key",
		StorageDriver:    "local",
		LocalStoragePath: tempDir,
	}, Services{
		Apps:      service.NewAppService(repo),
		Releases:  service.NewReleaseService(repo),
		Artifacts: service.NewArtifactService(repo, storage.NewLocalStorage(tempDir, "http://updates.example.test")),
		Metadata:  service.NewMetadataService(repo),
		Stats:     service.NewStatsService(repo),
		Auth:      service.NewAuthService(repo, tokens),
		Tokens:    tokens,
	})
	server := httptest.NewServer(router)

	login := postJSON[loginResponse](t, server, "/api/auth/login", map[string]string{
		"email":    "admin@example.com",
		"password": "secret-password",
	}, "")
	return server, "Bearer " + login.AccessToken
}

func newMockAnyshare(t *testing.T) (*httptest.Server, *anyshare.Client) {
	t.Helper()
	const sharingID = "AA121158B8D88B4E7C9019EA24FD02E541"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		baseURL := "http://" + r.Host
		switch r.URL.Path {
		case "/link/" + sharingID:
			w.Header().Add("Set-Cookie", "link_token:"+sharingID+"=anon-token; Path=/")
			w.WriteHeader(http.StatusOK)
		case "/api/efast/v1/file/osbeginupload":
			if r.Header.Get("Authorization") != "Bearer anon-token" {
				t.Fatalf("expected anyshare authorization on beginupload, got %q", r.Header.Get("Authorization"))
			}
			writeAnyshareJSON(w, map[string]any{
				"docid": "gns://anyshare-docid",
				"rev":   "anyshare-rev",
				"authrequest": []string{
					"POST",
					baseURL + "/upload",
					"AWSAccessKeyId: key",
					"Content-Type: application/octet-stream",
					"Policy: policy",
					"Signature: signature",
					"key: object-key",
				},
			})
		case "/upload":
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse anyshare upload: %v", err)
			}
			file, _, err := r.FormFile("file")
			if err != nil {
				t.Fatalf("expected anyshare upload file: %v", err)
			}
			_ = file.Close()
			w.WriteHeader(http.StatusNoContent)
		case "/api/efast/v1/file/osendupload":
			writeAnyshareJSON(w, map[string]any{"ok": true})
		case "/api/efast/v1/file/osdownload":
			if r.Header.Get("Authorization") != "Bearer anon-token" {
				t.Fatalf("expected anyshare authorization on osdownload, got %q", r.Header.Get("Authorization"))
			}
			writeAnyshareJSON(w, map[string]any{
				"size":        17,
				"authrequest": []string{"GET", baseURL + "/direct/desktop-app.exe?token=download"},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	client, err := anyshare.NewClient(context.Background(), anyshare.Config{
		BaseURL:     server.URL,
		SharingLink: server.URL + "/link/" + sharingID,
		UploadPath:  "gns://upload-dir",
		Timeout:     time.Second,
	})
	if err != nil {
		server.Close()
		t.Fatalf("create anyshare test client: %v", err)
	}
	return server, client
}

func writeAnyshareJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func findStagingClient(t *testing.T, server *httptest.Server, slug, version string) string {
	t.Helper()

	for i := 0; i < 1000; i++ {
		clientID := fmt.Sprintf("client-%03d", i)
		manifest := getJSON[domain.UpdateManifest](t, server, "/api/latest/"+slug+"/update.json?client_id="+clientID, "")
		if manifest.Version == version {
			return clientID
		}
	}
	t.Fatal("could not find staging client")
	return ""
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
	return uploadArtifactWithSource(t, server, releaseID, bearer, "")
}

func uploadArtifactWithSource(t *testing.T, server *httptest.Server, releaseID uuid.UUID, bearer, sourceType string) domain.Artifact {
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
	if sourceType != "" {
		if err := writer.WriteField("source_type", sourceType); err != nil {
			t.Fatalf("write source_type: %v", err)
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

func replaceArtifactFile(t *testing.T, server *httptest.Server, artifactID uuid.UUID, bearer string) domain.Artifact {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "desktop-app-v2.exe")
	if err != nil {
		t.Fatalf("create form file: %v", err)
	}
	if _, err := part.Write([]byte("replacement payload")); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req, err := http.NewRequest(http.MethodPut, server.URL+"/api/artifacts/"+artifactID.String()+"/file", &body)
	if err != nil {
		t.Fatalf("create replace request: %v", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", bearer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("replace artifact: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(resp.Body)
		t.Fatalf("replace got status %d: %s", resp.StatusCode, string(data))
	}

	var artifact domain.Artifact
	if err := json.NewDecoder(resp.Body).Decode(&artifact); err != nil {
		t.Fatalf("decode replaced artifact: %v", err)
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
	mu             sync.Mutex
	apps           map[uuid.UUID]domain.App
	appSlugs       map[string]uuid.UUID
	releases       map[uuid.UUID]domain.Release
	artifacts      map[uuid.UUID]domain.Artifact
	admins         map[string]domain.AdminUser
	updateEvents   []domain.UpdateRequestEvent
	downloadEvents []domain.DownloadEvent
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
		updateEvents:   make([]domain.UpdateRequestEvent, 0),
		downloadEvents: make([]domain.DownloadEvent, 0),
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
	app.UpdatedAt = app.CreatedAt
	r.apps[app.ID] = *app
	r.appSlugs[app.Slug] = app.ID
	return nil
}

func (r *memoryRepository) UpdateApp(ctx context.Context, app *domain.App) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	current, err := r.getAppByID(ctx, app.ID)
	if err != nil {
		return err
	}
	current.Name = app.Name
	current.Description = app.Description
	current.IconURL = app.IconURL
	current.DefaultChannel = app.DefaultChannel
	current.UpdatedAt = time.Now().UTC()
	r.apps[current.ID] = current
	*app = current
	return nil
}

func (r *memoryRepository) ArchiveApp(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	app, err := r.getAppByID(ctx, id)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	app.ArchivedAt = &now
	app.UpdatedAt = now
	r.apps[id] = app
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
		if app.ArchivedAt != nil {
			continue
		}
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
	app := r.apps[id]
	if app.ArchivedAt != nil {
		return domain.App{}, domain.ErrNotFound
	}
	return app, nil
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
	release.UpdatedAt = release.CreatedAt
	r.releases[release.ID] = *release
	return nil
}

func (r *memoryRepository) UpdateRelease(ctx context.Context, release *domain.Release) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	current, ok := r.releases[release.ID]
	if !ok || current.ArchivedAt != nil {
		return domain.ErrNotFound
	}
	current.Version = release.Version
	current.Channel = release.Channel
	current.Changelog = release.Changelog
	current.IsForced = release.IsForced
	current.StagingPercent = release.StagingPercent
	current.PublishedAt = release.PublishedAt
	current.UpdatedAt = time.Now().UTC()
	r.releases[current.ID] = current
	*release = current
	return nil
}

func (r *memoryRepository) ArchiveRelease(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	release, ok := r.releases[id]
	if !ok || release.ArchivedAt != nil {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	release.ArchivedAt = &now
	release.UpdatedAt = now
	r.releases[id] = release
	return nil
}

func (r *memoryRepository) GetReleaseByID(ctx context.Context, id uuid.UUID) (domain.Release, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.Release{}, err
	}
	release, ok := r.releases[id]
	if !ok || release.ArchivedAt != nil {
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
		if release.AppID == appID && release.ArchivedAt == nil {
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
		if release.AppID != appID || release.Channel != channel || release.PublishedAt.After(now) || release.ArchivedAt != nil {
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

func (r *memoryRepository) ListPublishedReleasesByAppChannel(ctx context.Context, appID uuid.UUID, channel string) ([]domain.Release, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	releases := make([]domain.Release, 0)
	for _, release := range r.releases {
		if release.AppID == appID && release.Channel == channel && !release.PublishedAt.After(now) && release.ArchivedAt == nil {
			releases = append(releases, release)
		}
	}
	sort.Slice(releases, func(i, j int) bool {
		if releases[i].PublishedAt.Equal(releases[j].PublishedAt) {
			return releases[i].CreatedAt.After(releases[j].CreatedAt)
		}
		return releases[i].PublishedAt.After(releases[j].PublishedAt)
	})
	return releases, nil
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
	artifact.UpdatedAt = artifact.CreatedAt
	r.artifacts[artifact.ID] = *artifact
	return nil
}

func (r *memoryRepository) UpdateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	current, ok := r.artifacts[artifact.ID]
	if !ok || current.ArchivedAt != nil {
		return domain.ErrNotFound
	}
	current.Platform = artifact.Platform
	current.Arch = artifact.Arch
	current.FileType = artifact.FileType
	current.FileName = artifact.FileName
	current.UpdatedAt = time.Now().UTC()
	r.artifacts[current.ID] = current
	*artifact = current
	return nil
}

func (r *memoryRepository) UpdateArtifactFile(ctx context.Context, artifact *domain.Artifact) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	current, ok := r.artifacts[artifact.ID]
	if !ok || current.ArchivedAt != nil {
		return domain.ErrNotFound
	}
	current.FileName = artifact.FileName
	current.FileURL = artifact.FileURL
	current.StorageKey = artifact.StorageKey
	current.FileSize = artifact.FileSize
	current.SHA512 = artifact.SHA512
	current.SourceType = artifact.SourceType
	current.AnyshareDocID = artifact.AnyshareDocID
	current.AnyshareRev = artifact.AnyshareRev
	current.AnyshareName = artifact.AnyshareName
	current.UpdatedAt = time.Now().UTC()
	r.artifacts[current.ID] = current
	*artifact = current
	return nil
}

func (r *memoryRepository) ArchiveArtifact(ctx context.Context, id uuid.UUID) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	artifact, ok := r.artifacts[id]
	if !ok || artifact.ArchivedAt != nil {
		return domain.ErrNotFound
	}
	now := time.Now().UTC()
	artifact.ArchivedAt = &now
	artifact.UpdatedAt = now
	r.artifacts[id] = artifact
	return nil
}

func (r *memoryRepository) GetArtifactByID(ctx context.Context, id uuid.UUID) (domain.Artifact, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.Artifact{}, err
	}
	artifact, ok := r.artifacts[id]
	if !ok || artifact.ArchivedAt != nil {
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
		if artifact.ReleaseID == releaseID && artifact.ArchivedAt == nil {
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

func (r *memoryRepository) CreateUpdateRequestEvent(ctx context.Context, event *domain.UpdateRequestEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	event.CreatedAt = time.Now().UTC()
	r.updateEvents = append(r.updateEvents, *event)
	return nil
}

func (r *memoryRepository) CreateDownloadEvent(ctx context.Context, event *domain.DownloadEvent) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}
	event.CreatedAt = time.Now().UTC()
	r.downloadEvents = append(r.downloadEvents, *event)
	return nil
}

func (r *memoryRepository) GetStatsSummary(ctx context.Context, filter domain.StatsFilter) (domain.StatsSummary, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return domain.StatsSummary{}, err
	}
	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	sevenDaysAgo := now.AddDate(0, 0, -7)

	summary := domain.StatsSummary{
		UpdateTrend:       make([]domain.StatsPoint, 0, 7),
		DownloadTrend:     make([]domain.StatsPoint, 0, 7),
		PlatformBreakdown: make([]domain.StatsBreakdown, 0),
		ArchBreakdown:     make([]domain.StatsBreakdown, 0),
		ChannelBreakdown:  make([]domain.StatsBreakdown, 0),
		VersionBreakdown:  make([]domain.StatsBreakdown, 0),
	}

	clients := make(map[string]struct{})
	platformCounts := make(map[string]int64)
	archCounts := make(map[string]int64)
	channelCounts := make(map[string]int64)
	versionCounts := make(map[string]int64)

	for _, event := range r.updateEvents {
		if !matchesUpdateStatsFilter(event, filter) {
			continue
		}
		summary.TotalUpdateRequests++
		if !event.CreatedAt.Before(today) {
			summary.TodayUpdateRequests++
		}
		if !event.CreatedAt.Before(sevenDaysAgo) {
			if event.ClientID != "" {
				clients[event.ClientID] = struct{}{}
			}
			if event.StagedHit {
				summary.StagingHits7d++
			}
			channelCounts[breakdownKey(event.Channel)]++
		}
	}
	for _, event := range r.downloadEvents {
		release := r.releases[event.ReleaseID]
		if !matchesDownloadStatsFilter(event, release, filter) {
			continue
		}
		summary.TotalDownloads++
		if !event.CreatedAt.Before(today) {
			summary.TodayDownloads++
		}
		if !event.CreatedAt.Before(sevenDaysAgo) {
			summary.Downloads7d++
			if event.ClientID != "" {
				clients[event.ClientID] = struct{}{}
			}
			platformCounts[breakdownKey(event.Platform)]++
			archCounts[breakdownKey(event.Arch)]++
			versionCounts[breakdownKey(event.Version)]++
			if release.IsForced {
				summary.ForcedDownloads7d++
			}
		}
	}
	summary.ActiveClients7d = int64(len(clients))
	summary.PlatformBreakdown = breakdownFromMap(platformCounts)
	summary.ArchBreakdown = breakdownFromMap(archCounts)
	summary.ChannelBreakdown = breakdownFromMap(channelCounts)
	summary.VersionBreakdown = breakdownFromMap(versionCounts)
	summary.UpdateTrend = trendFromUpdates(r.updateEvents, filter, now)
	summary.DownloadTrend = trendFromDownloads(r.downloadEvents, r.releases, filter, now)
	return summary, nil
}

func (r *memoryRepository) ListUpdateRequestEvents(ctx context.Context, filter domain.StatsFilter) ([]domain.UpdateRequestEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events := make([]domain.UpdateRequestEvent, 0)
	for _, event := range r.updateEvents {
		if !matchesUpdateStatsFilter(event, filter) {
			continue
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	events = applyStatsWindow(events, filter.Limit, filter.Offset)
	return events, nil
}

func (r *memoryRepository) ListDownloadEvents(ctx context.Context, filter domain.StatsFilter) ([]domain.DownloadEvent, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	events := make([]domain.DownloadEvent, 0)
	for _, event := range r.downloadEvents {
		release := r.releases[event.ReleaseID]
		if !matchesDownloadStatsFilter(event, release, filter) {
			continue
		}
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt.After(events[j].CreatedAt) })
	events = applyStatsWindow(events, filter.Limit, filter.Offset)
	return events, nil
}

func matchesUpdateStatsFilter(event domain.UpdateRequestEvent, filter domain.StatsFilter) bool {
	if filter.AppID != nil && (event.AppID == nil || *event.AppID != *filter.AppID) {
		return false
	}
	if filter.ReleaseID != nil && (event.ReleaseID == nil || *event.ReleaseID != *filter.ReleaseID) {
		return false
	}
	return matchesCommonStatsFilter(event.CreatedAt, event.Channel, event.Platform, event.Arch, event.ClientID, filter)
}

func matchesDownloadStatsFilter(event domain.DownloadEvent, release domain.Release, filter domain.StatsFilter) bool {
	if filter.AppID != nil && event.AppID != *filter.AppID {
		return false
	}
	if filter.ReleaseID != nil && event.ReleaseID != *filter.ReleaseID {
		return false
	}
	return matchesCommonStatsFilter(event.CreatedAt, release.Channel, event.Platform, event.Arch, event.ClientID, filter)
}

func matchesCommonStatsFilter(createdAt time.Time, channel, platform, arch, clientID string, filter domain.StatsFilter) bool {
	if filter.DateFrom != nil && createdAt.Before(*filter.DateFrom) {
		return false
	}
	if filter.DateTo != nil && createdAt.After(*filter.DateTo) {
		return false
	}
	if filter.Channel != "" && channel != filter.Channel {
		return false
	}
	if filter.Platform != "" && platform != filter.Platform {
		return false
	}
	if filter.Arch != "" && arch != filter.Arch {
		return false
	}
	if filter.ClientID != "" && clientID != filter.ClientID {
		return false
	}
	return true
}

func applyStatsWindow[T any](items []T, limit, offset int) []T {
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []T{}
	}
	items = items[offset:]
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func (r *memoryRepository) getAppByID(ctx context.Context, id uuid.UUID) (domain.App, error) {
	if err := ctx.Err(); err != nil {
		return domain.App{}, err
	}
	app, ok := r.apps[id]
	if !ok || app.ArchivedAt != nil {
		return domain.App{}, domain.ErrNotFound
	}
	return app, nil
}

func breakdownKey(value string) string {
	if strings.TrimSpace(value) == "" {
		return "unknown"
	}
	return value
}

func breakdownFromMap(values map[string]int64) []domain.StatsBreakdown {
	items := make([]domain.StatsBreakdown, 0, len(values))
	for key, count := range values {
		items = append(items, domain.StatsBreakdown{Key: key, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Key < items[j].Key
		}
		return items[i].Count > items[j].Count
	})
	return items
}

func trendFromUpdates(events []domain.UpdateRequestEvent, filter domain.StatsFilter, now time.Time) []domain.StatsPoint {
	counts := make(map[string]int64)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
	for _, event := range events {
		if !matchesUpdateStatsFilter(event, filter) {
			continue
		}
		if event.CreatedAt.Before(start) {
			continue
		}
		counts[event.CreatedAt.Format("2006-01-02")]++
	}
	return buildTrend(start, counts)
}

func trendFromDownloads(events []domain.DownloadEvent, releases map[uuid.UUID]domain.Release, filter domain.StatsFilter, now time.Time) []domain.StatsPoint {
	counts := make(map[string]int64)
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, -6)
	for _, event := range events {
		if !matchesDownloadStatsFilter(event, releases[event.ReleaseID], filter) {
			continue
		}
		if event.CreatedAt.Before(start) {
			continue
		}
		counts[event.CreatedAt.Format("2006-01-02")]++
	}
	return buildTrend(start, counts)
}

func buildTrend(start time.Time, counts map[string]int64) []domain.StatsPoint {
	points := make([]domain.StatsPoint, 0, 7)
	for i := 0; i < 7; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		points = append(points, domain.StatsPoint{Date: date, Count: counts[date]})
	}
	return points
}
