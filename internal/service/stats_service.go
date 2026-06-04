package service

import (
	"context"
	"strings"

	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
)

type StatsService struct {
	repo Repository
}

type RequestMeta struct {
	IP        string
	UserAgent string
	ClientID  string
}

type RecordUpdateInput struct {
	AppSlug   string
	App       *domain.App
	Release   *domain.Release
	Query     LatestQuery
	Format    string
	Matched   bool
	StagedHit bool
	Meta      RequestMeta
}

func NewStatsService(repo Repository) *StatsService {
	return &StatsService{repo: repo}
}

func (s *StatsService) RecordUpdateRequest(ctx context.Context, input RecordUpdateInput) error {
	event := domain.UpdateRequestEvent{
		ID:        uuid.New(),
		AppSlug:   strings.TrimSpace(input.AppSlug),
		Channel:   normalizeChannel(input.Query.Channel),
		Platform:  normalizeToken(input.Query.Platform),
		Arch:      normalizeToken(input.Query.Arch),
		ClientID:  strings.TrimSpace(input.Query.ClientID),
		Format:    normalizeFormat(input.Format),
		Matched:   input.Matched,
		StagedHit: input.StagedHit,
		IP:        strings.TrimSpace(input.Meta.IP),
		UserAgent: strings.TrimSpace(input.Meta.UserAgent),
	}
	if input.App != nil {
		event.AppID = &input.App.ID
		if input.Query.Channel == "" {
			event.Channel = input.App.DefaultChannel
		}
	}
	if input.Release != nil {
		event.ReleaseID = &input.Release.ID
		event.Version = input.Release.Version
		event.Channel = input.Release.Channel
	}
	return s.repo.CreateUpdateRequestEvent(ctx, &event)
}

func (s *StatsService) RecordDownload(ctx context.Context, artifact domain.Artifact, meta RequestMeta) error {
	release, err := s.repo.GetReleaseByID(ctx, artifact.ReleaseID)
	if err != nil {
		return err
	}
	event := domain.DownloadEvent{
		ID:         uuid.New(),
		AppID:      release.AppID,
		ReleaseID:  release.ID,
		ArtifactID: artifact.ID,
		Version:    release.Version,
		Platform:   artifact.Platform,
		Arch:       artifact.Arch,
		FileType:   artifact.FileType,
		FileName:   artifact.FileName,
		ClientID:   strings.TrimSpace(meta.ClientID),
		IP:         strings.TrimSpace(meta.IP),
		UserAgent:  strings.TrimSpace(meta.UserAgent),
	}
	return s.repo.CreateDownloadEvent(ctx, &event)
}

func (s *StatsService) Summary(ctx context.Context, filter domain.StatsFilter) (domain.StatsSummary, error) {
	filter = normalizeStatsFilter(filter)
	return s.repo.GetStatsSummary(ctx, filter)
}

func (s *StatsService) RecentUpdates(ctx context.Context, filter domain.StatsFilter) ([]domain.UpdateRequestEvent, error) {
	filter = normalizeStatsFilter(filter)
	return s.repo.ListUpdateRequestEvents(ctx, filter)
}

func (s *StatsService) RecentDownloads(ctx context.Context, filter domain.StatsFilter) ([]domain.DownloadEvent, error) {
	filter = normalizeStatsFilter(filter)
	return s.repo.ListDownloadEvents(ctx, filter)
}

func normalizeStatsFilter(filter domain.StatsFilter) domain.StatsFilter {
	filter.Channel = normalizeChannel(filter.Channel)
	filter.Platform = normalizeToken(filter.Platform)
	filter.Arch = normalizeToken(filter.Arch)
	filter.ClientID = strings.TrimSpace(filter.ClientID)
	filter.Limit = normalizeLimit(filter.Limit)
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return filter
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 50
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "yaml", "yml", "latest.yml":
		return "yml"
	case "xml", "appcast.xml":
		return "xml"
	default:
		return "json"
	}
}
