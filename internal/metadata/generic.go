package metadata

import (
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
			URL:      artifact.FileURL,
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
