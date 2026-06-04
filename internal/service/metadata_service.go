package service

import (
	"context"
	"hash/fnv"

	"multi-platform-distribution/internal/domain"
	"multi-platform-distribution/internal/metadata"
)

type MetadataService struct {
	repo Repository
}

type LatestResult struct {
	Manifest  domain.UpdateManifest
	App       domain.App
	Release   domain.Release
	StagedHit bool
}

type LatestQuery struct {
	Channel  string
	Platform string
	Arch     string
	ClientID string
}

func NewMetadataService(repo Repository) *MetadataService {
	return &MetadataService{repo: repo}
}

func (s *MetadataService) Latest(ctx context.Context, appSlug string, query LatestQuery) (domain.UpdateManifest, error) {
	result, err := s.LatestWithDetails(ctx, appSlug, query)
	if err != nil {
		return domain.UpdateManifest{}, err
	}
	return result.Manifest, nil
}

func (s *MetadataService) LatestWithDetails(ctx context.Context, appSlug string, query LatestQuery) (LatestResult, error) {
	app, err := s.repo.GetAppBySlug(ctx, appSlug)
	if err != nil {
		return LatestResult{}, err
	}

	channel := normalizeChannel(query.Channel)
	if query.Channel == "" {
		channel = app.DefaultChannel
	}

	releases, err := s.repo.ListPublishedReleasesByAppChannel(ctx, app.ID, channel)
	if err != nil {
		return LatestResult{}, err
	}
	release, stagedHit, err := chooseReleaseForClient(app.Slug, releases, query.ClientID)
	if err != nil {
		return LatestResult{}, err
	}

	artifacts, err := s.repo.ListArtifactsByRelease(ctx, release.ID)
	if err != nil {
		return LatestResult{}, err
	}

	platform := normalizeToken(query.Platform)
	arch := normalizeToken(query.Arch)

	return LatestResult{
		Manifest:  metadata.BuildGeneric(release, artifacts, platform, arch),
		App:       app,
		Release:   release,
		StagedHit: stagedHit,
	}, nil
}

func chooseReleaseForClient(appSlug string, releases []domain.Release, clientID string) (domain.Release, bool, error) {
	for _, release := range releases {
		if release.StagingPercent >= 100 {
			return release, false, nil
		}
		if release.StagingPercent <= 0 || clientID == "" {
			continue
		}
		if stagingBucket(appSlug, release.Channel, release.Version, clientID) < release.StagingPercent {
			return release, true, nil
		}
	}
	return domain.Release{}, false, domain.ErrNotFound
}

func stagingBucket(appSlug, channel, version, clientID string) int {
	hash := fnv.New32a()
	_, _ = hash.Write([]byte(appSlug + ":" + channel + ":" + version + ":" + clientID))
	return int(hash.Sum32() % 100)
}
