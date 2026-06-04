package metadata

import (
	"strings"
	"testing"
	"time"

	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
)

func TestBuildGenericFiltersArtifacts(t *testing.T) {
	release := domain.Release{
		ID:             uuid.New(),
		Version:        "1.2.3",
		Channel:        "stable",
		Changelog:      "ship it",
		IsForced:       true,
		StagingPercent: 50,
		PublishedAt:    time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
	}
	artifacts := []domain.Artifact{
		{
			Platform: "windows",
			Arch:     "x64",
			FileType: "exe",
			FileURL:  "https://example.com/app.exe",
			FileSize: 123,
			SHA512:   "abc",
		},
		{
			Platform: "macos",
			Arch:     "arm64",
			FileType: "dmg",
			FileURL:  "https://example.com/app.dmg",
			FileSize: 456,
			SHA512:   "def",
		},
	}

	manifest := BuildGeneric(release, artifacts, "windows", "x64")

	if manifest.Version != "1.2.3" {
		t.Fatalf("expected version 1.2.3, got %s", manifest.Version)
	}
	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(manifest.Files))
	}
	if manifest.Files[0].URL != "https://example.com/app.exe" {
		t.Fatalf("unexpected file URL %s", manifest.Files[0].URL)
	}
}

func TestBuildGenericAddsFilenameToAnyshareDownloadURL(t *testing.T) {
	release := domain.Release{
		ID:          uuid.New(),
		Version:     "1.2.3",
		Channel:     "stable",
		PublishedAt: time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
	}
	artifacts := []domain.Artifact{
		{
			Platform:      "windows",
			Arch:          "x64",
			FileType:      "exe",
			FileName:      "AutoUp LittleTools Setup 0.1.3.exe",
			FileURL:       "https://updates.example.test/api/artifacts/5963f2c8/download",
			FileSize:      123,
			SHA512:        "abc",
			SourceType:    "anyshare",
			AnyshareName:  "AutoUp LittleTools Setup 0.1.3.exe",
			AnyshareDocID: "gns://file",
		},
	}

	manifest := BuildGeneric(release, artifacts, "windows", "x64")

	if len(manifest.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(manifest.Files))
	}
	expected := "https://updates.example.test/api/artifacts/5963f2c8/download/AutoUp%20LittleTools%20Setup%200.1.3.exe"
	if manifest.Files[0].URL != expected {
		t.Fatalf("expected anyshare URL %s, got %s", expected, manifest.Files[0].URL)
	}
}

func TestRenderElectronLatest(t *testing.T) {
	manifest := sampleManifest()

	body, err := RenderElectronLatest(manifest)
	if err != nil {
		t.Fatalf("render electron latest: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, "version: 1.2.3") {
		t.Fatalf("expected version in latest.yml, got %s", text)
	}
	if !strings.Contains(text, "sha512: abc") {
		t.Fatalf("expected sha512 in latest.yml, got %s", text)
	}
	if !strings.Contains(text, "releaseNotes: ship it") || !strings.Contains(text, "changelog: ship it") {
		t.Fatalf("expected changelog in latest.yml, got %s", text)
	}
}

func TestRenderAppcast(t *testing.T) {
	manifest := sampleManifest()

	body, err := RenderAppcast("myapp", manifest)
	if err != nil {
		t.Fatalf("render appcast: %v", err)
	}
	text := string(body)
	if !strings.Contains(text, `<rss version="2.0"`) {
		t.Fatalf("expected rss root, got %s", text)
	}
	if !strings.Contains(text, `sparkle:version="1.2.3"`) {
		t.Fatalf("expected sparkle version, got %s", text)
	}
}

func sampleManifest() domain.UpdateManifest {
	return domain.UpdateManifest{
		Version:        "1.2.3",
		Channel:        "stable",
		Changelog:      "ship it",
		Forced:         true,
		StagingPercent: 50,
		PublishedAt:    time.Date(2026, 6, 3, 0, 0, 0, 0, time.UTC),
		Files: []domain.UpdatePackage{
			{
				Platform: "windows",
				Arch:     "x64",
				Type:     "exe",
				URL:      "https://example.com/app.exe",
				Size:     123,
				SHA512:   "abc",
			},
		},
	}
}
