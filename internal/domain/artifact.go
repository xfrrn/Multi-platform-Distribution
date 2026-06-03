package domain

import (
	"time"

	"github.com/google/uuid"
)

type Artifact struct {
	ID         uuid.UUID `json:"id"`
	ReleaseID  uuid.UUID `json:"release_id"`
	Platform   string    `json:"platform"`
	Arch       string    `json:"arch"`
	FileType   string    `json:"file_type"`
	FileURL    string    `json:"file_url"`
	StorageKey string    `json:"storage_key"`
	FileSize   int64     `json:"file_size"`
	SHA512     string    `json:"sha512"`
	CreatedAt  time.Time `json:"created_at"`
}
