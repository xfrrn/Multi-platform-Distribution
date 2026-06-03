package service

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
)

type ArtifactService struct {
	repo    Repository
	storage ObjectStorage
}

type UploadArtifactInput struct {
	ReleaseID uuid.UUID
	Platform  string
	Arch      string
	FileType  string
	FileName  string
	Body      io.Reader
}

func NewArtifactService(repo Repository, storage ObjectStorage) *ArtifactService {
	return &ArtifactService{repo: repo, storage: storage}
}

func (s *ArtifactService) Upload(ctx context.Context, input UploadArtifactInput) (domain.Artifact, error) {
	if input.Body == nil {
		return domain.Artifact{}, errors.New("file is required")
	}
	platform := normalizeToken(input.Platform)
	arch := normalizeToken(input.Arch)
	fileType := strings.TrimPrefix(normalizeToken(input.FileType), ".")
	if platform == "" || arch == "" || fileType == "" {
		return domain.Artifact{}, errors.New("platform, arch, and file_type are required")
	}

	release, err := s.repo.GetReleaseByID(ctx, input.ReleaseID)
	if err != nil {
		return domain.Artifact{}, err
	}

	name := safeFilename(input.FileName)
	if name == "" {
		name = "artifact." + fileType
	}
	key := strings.Join([]string{
		release.AppID.String(),
		release.ID.String(),
		platform,
		arch,
		name,
	}, "/")

	obj, err := s.storage.Save(ctx, key, input.Body)
	if err != nil {
		return domain.Artifact{}, err
	}

	artifact := domain.Artifact{
		ID:         uuid.New(),
		ReleaseID:  release.ID,
		Platform:   platform,
		Arch:       arch,
		FileType:   fileType,
		FileURL:    obj.URL,
		StorageKey: obj.Key,
		FileSize:   obj.Size,
		SHA512:     obj.SHA512,
	}
	if err := s.repo.CreateArtifact(ctx, &artifact); err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s *ArtifactService) Get(ctx context.Context, id uuid.UUID) (domain.Artifact, error) {
	return s.repo.GetArtifactByID(ctx, id)
}

func (s *ArtifactService) ListByRelease(ctx context.Context, releaseID uuid.UUID) ([]domain.Artifact, error) {
	return s.repo.ListArtifactsByRelease(ctx, releaseID)
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "\\", "")
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}
