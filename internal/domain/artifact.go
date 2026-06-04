package domain

import (
	"time"

	"github.com/google/uuid"
)

type Artifact struct {
	ID             uuid.UUID  `json:"id"`
	ReleaseID      uuid.UUID  `json:"release_id"`
	Platform       string     `json:"platform"`
	Arch           string     `json:"arch"`
	FileType       string     `json:"file_type"`
	FileName       string     `json:"file_name"`
	FileURL        string     `json:"file_url"`
	StorageKey     string     `json:"storage_key"`
	FileSize       int64      `json:"file_size"`
	SHA512         string     `json:"sha512"`
	SourceType     string     `json:"source_type"`
	AnyshareDocID  string     `json:"anyshare_docid,omitempty"`
	AnyshareRev    string     `json:"anyshare_rev,omitempty"`
	AnyshareName   string     `json:"anyshare_name,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
}
