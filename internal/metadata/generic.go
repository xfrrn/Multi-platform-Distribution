package metadata

import (
	"net/url"
	"path"
	"path/filepath"
	"strings"

	"multi-platform-distribution/internal/domain"
)

func BuildGeneric(release domain.Release, artifacts []domain.Artifact, platform, arch string) domain.UpdateManifest {
	files := make([]domain.UpdatePackage, 0, len(artifacts))
	for _, artifact := range artifacts {
		if platform != "" && artifact.Platform != platform {
			continue
		}
		if arch != "" && artifact.Arch != arch {
			continue
		}
		files = append(files, domain.UpdatePackage{
			Platform: artifact.Platform,
			Arch:     artifact.Arch,
			Type:     artifact.FileType,
			URL:      updatePackageURL(artifact),
			Size:     artifact.FileSize,
			SHA512:   artifact.SHA512,
		})
	}

	return domain.UpdateManifest{
		Version:        release.Version,
		Channel:        release.Channel,
		Changelog:      release.Changelog,
		Forced:         release.IsForced,
		StagingPercent: release.StagingPercent,
		PublishedAt:    release.PublishedAt,
		Files:          files,
	}
}

func updatePackageURL(artifact domain.Artifact) string {
	if strings.ToLower(strings.TrimSpace(artifact.SourceType)) != "anyshare" {
		return artifact.FileURL
	}

	name := safeURLFilename(artifact.AnyshareName)
	if name == "" {
		name = safeURLFilename(artifact.FileName)
	}
	if name == "" {
		return artifact.FileURL
	}

	parsed, err := url.Parse(artifact.FileURL)
	if err != nil {
		return artifact.FileURL
	}
	if !strings.HasSuffix(strings.TrimRight(parsed.Path, "/"), "/download") {
		return artifact.FileURL
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/" + name
	if parsed.Scheme == "" && parsed.Host == "" {
		return parsed.String()
	}
	return parsed.String()
}

func safeURLFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "\\", "")
	if name == "." || name == string(filepath.Separator) || name == "" {
		return ""
	}
	return path.Base(name)
}
