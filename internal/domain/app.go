package domain

import (
	"time"

	"github.com/google/uuid"
)

type App struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Slug           string     `json:"slug"`
	Description    string     `json:"description"`
	IconURL        string     `json:"icon_url"`
	DefaultChannel string     `json:"default_channel"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ArchivedAt     *time.Time `json:"archived_at,omitempty"`
}
