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

type UpdateArtifactInput struct {
	Platform string `json:"platform"`
	Arch     string `json:"arch"`
	FileType string `json:"file_type"`
	FileName string `json:"file_name"`
}

type ReplaceArtifactFileInput struct {
	FileName string
	Body     io.Reader
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
		FileName:   name,
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
	if _, err := s.repo.GetReleaseByID(ctx, releaseID); err != nil {
		return nil, err
	}
	return s.repo.ListArtifactsByRelease(ctx, releaseID)
}

func (s *ArtifactService) Update(ctx context.Context, id uuid.UUID, input UpdateArtifactInput) (domain.Artifact, error) {
	artifact, err := s.repo.GetArtifactByID(ctx, id)
	if err != nil {
		return domain.Artifact{}, err
	}
	platform := normalizeToken(input.Platform)
	arch := normalizeToken(input.Arch)
	fileType := strings.TrimPrefix(normalizeToken(input.FileType), ".")
	fileName := safeFilename(input.FileName)
	if platform == "" || arch == "" || fileType == "" {
		return domain.Artifact{}, errors.New("platform, arch, and file_type are required")
	}
	if fileName == "" {
		fileName = artifact.FileName
	}

	artifact.Platform = platform
	artifact.Arch = arch
	artifact.FileType = fileType
	artifact.FileName = fileName
	if err := s.repo.UpdateArtifact(ctx, &artifact); err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s *ArtifactService) ReplaceFile(ctx context.Context, id uuid.UUID, input ReplaceArtifactFileInput) (domain.Artifact, error) {
	if input.Body == nil {
		return domain.Artifact{}, errors.New("file is required")
	}
	artifact, err := s.repo.GetArtifactByID(ctx, id)
	if err != nil {
		return domain.Artifact{}, err
	}
	release, err := s.repo.GetReleaseByID(ctx, artifact.ReleaseID)
	if err != nil {
		return domain.Artifact{}, err
	}
	fileName := safeFilename(input.FileName)
	if fileName == "" {
		fileName = artifact.FileName
	}
	key := strings.Join([]string{
		release.AppID.String(),
		release.ID.String(),
		artifact.Platform,
		artifact.Arch,
		fileName,
	}, "/")
	obj, err := s.storage.Save(ctx, key, input.Body)
	if err != nil {
		return domain.Artifact{}, err
	}
	artifact.FileName = fileName
	artifact.FileURL = obj.URL
	artifact.StorageKey = obj.Key
	artifact.FileSize = obj.Size
	artifact.SHA512 = obj.SHA512
	if err := s.repo.UpdateArtifactFile(ctx, &artifact); err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s *ArtifactService) Archive(ctx context.Context, id uuid.UUID) error {
	return s.repo.ArchiveArtifact(ctx, id)
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
