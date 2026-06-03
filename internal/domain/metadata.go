package domain

import "time"

type UpdateManifest struct {
	Version        string          `json:"version"`
	Channel        string          `json:"channel"`
	Changelog      string          `json:"changelog"`
	Forced         bool            `json:"forced"`
	StagingPercent int             `json:"staging_percent"`
	PublishedAt    time.Time       `json:"published_at"`
	Files          []UpdatePackage `json:"files"`
}

type UpdatePackage struct {
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	SHA512   string `json:"sha512"`
}
