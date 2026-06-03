package service

import (
	"context"

	"multi-platform-distribution/internal/domain"
	"multi-platform-distribution/internal/metadata"
)

type MetadataService struct {
	repo Repository
}

type LatestQuery struct {
	Channel  string
	Platform string
	Arch     string
}

func NewMetadataService(repo Repository) *MetadataService {
	return &MetadataService{repo: repo}
}

func (s *MetadataService) Latest(ctx context.Context, appSlug string, query LatestQuery) (domain.UpdateManifest, error) {
	app, err := s.repo.GetAppBySlug(ctx, appSlug)
	if err != nil {
		return domain.UpdateManifest{}, err
	}

	channel := normalizeChannel(query.Channel)
	if query.Channel == "" {
		channel = app.DefaultChannel
	}

	release, err := s.repo.GetLatestRelease(ctx, app.ID, channel)
	if err != nil {
		return domain.UpdateManifest{}, err
	}

	artifacts, err := s.repo.ListArtifactsByRelease(ctx, release.ID)
	if err != nil {
		return domain.UpdateManifest{}, err
	}

	platform := normalizeToken(query.Platform)
	arch := normalizeToken(query.Arch)

	return metadata.BuildGeneric(release, artifacts, platform, arch), nil
}
