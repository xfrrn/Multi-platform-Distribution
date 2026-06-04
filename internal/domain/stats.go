package domain

import (
	"time"

	"github.com/google/uuid"
)

type UpdateRequestEvent struct {
	ID        uuid.UUID  `json:"id"`
	AppID     *uuid.UUID `json:"app_id,omitempty"`
	AppSlug   string     `json:"app_slug"`
	ReleaseID *uuid.UUID `json:"release_id,omitempty"`
	Version   string     `json:"version"`
	Channel   string     `json:"channel"`
	Platform  string     `json:"platform"`
	Arch      string     `json:"arch"`
	ClientID  string     `json:"client_id"`
	Format    string     `json:"format"`
	Matched   bool       `json:"matched"`
	StagedHit bool       `json:"staged_hit"`
	IP        string     `json:"ip"`
	UserAgent string     `json:"user_agent"`
	CreatedAt time.Time  `json:"created_at"`
}

type DownloadEvent struct {
	ID         uuid.UUID `json:"id"`
	AppID      uuid.UUID `json:"app_id"`
	ReleaseID  uuid.UUID `json:"release_id"`
	ArtifactID uuid.UUID `json:"artifact_id"`
	Version    string    `json:"version"`
	Platform   string    `json:"platform"`
	Arch       string    `json:"arch"`
	FileType   string    `json:"file_type"`
	FileName   string    `json:"file_name"`
	ClientID   string    `json:"client_id"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
}

type StatsSummary struct {
	TodayUpdateRequests int64            `json:"today_update_requests"`
	TodayDownloads      int64            `json:"today_downloads"`
	Downloads7d         int64            `json:"downloads_7d"`
	ActiveClients7d     int64            `json:"active_clients_7d"`
	StagingHits7d       int64            `json:"staging_hits_7d"`
	ForcedDownloads7d   int64            `json:"forced_downloads_7d"`
	TotalUpdateRequests int64            `json:"total_update_requests"`
	TotalDownloads      int64            `json:"total_downloads"`
	UpdateTrend         []StatsPoint     `json:"update_trend"`
	DownloadTrend       []StatsPoint     `json:"download_trend"`
	PlatformBreakdown   []StatsBreakdown `json:"platform_breakdown"`
	ArchBreakdown       []StatsBreakdown `json:"arch_breakdown"`
	ChannelBreakdown    []StatsBreakdown `json:"channel_breakdown"`
	VersionBreakdown    []StatsBreakdown `json:"version_breakdown"`
}

type StatsPoint struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type StatsBreakdown struct {
	Key   string `json:"key"`
	Count int64  `json:"count"`
}

type StatsFilter struct {
	AppID     *uuid.UUID
	ReleaseID *uuid.UUID
	DateFrom  *time.Time
	DateTo    *time.Time
	Channel   string
	Platform  string
	Arch      string
	ClientID  string
	Limit     int
	Offset    int
}
