package service

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"

	"multi-platform-distribution/internal/anyshare"
	"multi-platform-distribution/internal/domain"

	"github.com/google/uuid"
)

type ArtifactService struct {
	repo          Repository
	storage       ObjectStorage
	anyshare      *anyshare.Client
	publicBaseURL string
}

type UploadArtifactInput struct {
	ReleaseID  uuid.UUID
	Platform   string
	Arch       string
	FileType   string
	FileName   string
	SourceType string
	Body       io.Reader
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

func NewArtifactServiceWithAnyshare(repo Repository, storage ObjectStorage, anyshareClient *anyshare.Client, publicBaseURL string) *ArtifactService {
	return &ArtifactService{
		repo:          repo,
		storage:       storage,
		anyshare:      anyshareClient,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
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
	sourceType, err := normalizeSourceType(input.SourceType)
	if err != nil {
		return domain.Artifact{}, err
	}
	if sourceType == "anyshare" {
		return s.uploadAnyshare(context.WithoutCancel(ctx), release.ID, platform, arch, fileType, name, input.Body)
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
		SourceType: "managed",
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
	if normalizeArtifactSource(artifact.SourceType) == "anyshare" {
		if s.anyshare == nil {
			return domain.Artifact{}, errors.New("anyshare storage is not enabled")
		}
		uploadCtx := context.WithoutCancel(ctx)
		obj, err := s.anyshare.Upload(uploadCtx, fileName, input.Body)
		if err != nil {
			return domain.Artifact{}, err
		}
		artifact.FileName = fileName
		artifact.StorageKey = obj.DocID
		artifact.FileSize = obj.Size
		artifact.SHA512 = obj.SHA512
		artifact.SourceType = "anyshare"
		artifact.AnyshareDocID = obj.DocID
		artifact.AnyshareRev = obj.Rev
		artifact.AnyshareName = obj.Name
		if artifact.FileURL == "" {
			artifact.FileURL = s.artifactDownloadURL(artifact.ID)
		}
		if err := s.repo.UpdateArtifactFile(uploadCtx, &artifact); err != nil {
			return domain.Artifact{}, err
		}
		return artifact, nil
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
	artifact.SourceType = "managed"
	if err := s.repo.UpdateArtifactFile(ctx, &artifact); err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s *ArtifactService) Archive(ctx context.Context, id uuid.UUID) error {
	return s.repo.ArchiveArtifact(ctx, id)
}

func (s *ArtifactService) DownloadURL(ctx context.Context, artifact domain.Artifact) (string, error) {
	if normalizeArtifactSource(artifact.SourceType) != "anyshare" {
		return artifact.FileURL, nil
	}
	if s.anyshare == nil {
		return "", errors.New("anyshare storage is not enabled")
	}
	docID := artifact.AnyshareDocID
	if docID == "" {
		docID = artifact.StorageKey
	}
	result, err := s.anyshare.ResolveDownload(ctx, docID, artifact.AnyshareRev, artifact.AnyshareName)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

func (s *ArtifactService) uploadAnyshare(ctx context.Context, releaseID uuid.UUID, platform, arch, fileType, name string, body io.Reader) (domain.Artifact, error) {
	if s.anyshare == nil {
		return domain.Artifact{}, errors.New("anyshare storage is not enabled")
	}
	obj, err := s.anyshare.Upload(ctx, name, body)
	if err != nil {
		return domain.Artifact{}, err
	}
	artifactID := uuid.New()
	artifact := domain.Artifact{
		ID:            artifactID,
		ReleaseID:     releaseID,
		Platform:      platform,
		Arch:          arch,
		FileType:      fileType,
		FileName:      name,
		FileURL:       s.artifactDownloadURL(artifactID),
		StorageKey:    obj.DocID,
		FileSize:      obj.Size,
		SHA512:        obj.SHA512,
		SourceType:    "anyshare",
		AnyshareDocID: obj.DocID,
		AnyshareRev:   obj.Rev,
		AnyshareName:  obj.Name,
	}
	if err := s.repo.CreateArtifact(ctx, &artifact); err != nil {
		return domain.Artifact{}, err
	}
	return artifact, nil
}

func (s *ArtifactService) artifactDownloadURL(id uuid.UUID) string {
	if s.publicBaseURL == "" {
		return "/api/artifacts/" + id.String() + "/download"
	}
	return s.publicBaseURL + "/api/artifacts/" + id.String() + "/download"
}

func normalizeToken(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizeSourceType(value string) (string, error) {
	sourceType := normalizeArtifactSource(value)
	switch sourceType {
	case "managed", "anyshare":
		return sourceType, nil
	default:
		return "", errors.New("source_type must be managed or anyshare")
	}
}

func normalizeArtifactSource(value string) string {
	sourceType := strings.ToLower(strings.TrimSpace(value))
	if sourceType == "" {
		return "managed"
	}
	return sourceType
}

func safeFilename(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "\\", "")
	if name == "." || name == string(filepath.Separator) {
		return ""
	}
	return name
}
