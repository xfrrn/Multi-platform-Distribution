package metadata

import (
	"time"

	"multi-platform-distribution/internal/domain"

	"github.com/goccy/go-yaml"
)

type ElectronLatest struct {
	Version     string         `yaml:"version"`
	Files       []ElectronFile `yaml:"files"`
	Path        string         `yaml:"path,omitempty"`
	SHA512      string         `yaml:"sha512,omitempty"`
	ReleaseDate string         `yaml:"releaseDate"`
}

type ElectronFile struct {
	URL    string `yaml:"url"`
	SHA512 string `yaml:"sha512"`
	Size   int64  `yaml:"size"`
}

func RenderElectronLatest(manifest domain.UpdateManifest) ([]byte, error) {
	latest := ElectronLatest{
		Version:     manifest.Version,
		Files:       make([]ElectronFile, 0, len(manifest.Files)),
		ReleaseDate: manifest.PublishedAt.UTC().Format(time.RFC3339),
	}

	for _, file := range manifest.Files {
		latest.Files = append(latest.Files, ElectronFile{
			URL:    file.URL,
			SHA512: file.SHA512,
			Size:   file.Size,
		})
	}
	if len(latest.Files) > 0 {
		latest.Path = latest.Files[0].URL
		latest.SHA512 = latest.Files[0].SHA512
	}

	return yaml.Marshal(latest)
}
