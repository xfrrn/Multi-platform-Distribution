package domain

import (
	"time"

	"github.com/google/uuid"
)

type Release struct {
	ID             uuid.UUID `json:"id"`
	AppID          uuid.UUID `json:"app_id"`
	Version        string    `json:"version"`
	Channel        string    `json:"channel"`
	Changelog      string    `json:"changelog"`
	IsForced       bool      `json:"is_forced"`
	StagingPercent int       `json:"staging_percent"`
	PublishedAt    time.Time `json:"published_at"`
	CreatedAt      time.Time `json:"created_at"`
}
