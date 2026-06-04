package handlers

import (
	"net/http"

	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ArtifactHandler struct {
	artifacts *service.ArtifactService
	stats     *service.StatsService
}

func NewArtifactHandler(artifacts *service.ArtifactService, stats *service.StatsService) *ArtifactHandler {
	return &ArtifactHandler{artifacts: artifacts, stats: stats}
}

func (h *ArtifactHandler) Upload(c *gin.Context) {
	releaseID, err := uuid.Parse(c.Param("releaseId"))
	if err != nil {
		writeError(c, err)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		writeError(c, err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer file.Close()

	artifact, err := h.artifacts.Upload(c.Request.Context(), service.UploadArtifactInput{
		ReleaseID: releaseID,
		Platform:  c.PostForm("platform"),
		Arch:      c.PostForm("arch"),
		FileType:  c.PostForm("file_type"),
		FileName:  fileHeader.Filename,
		Body:      file,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, artifact)
}

func (h *ArtifactHandler) Get(c *gin.Context) {
	artifactID, err := uuid.Parse(c.Param("artifactId"))
	if err != nil {
		writeError(c, err)
		return
	}
	artifact, err := h.artifacts.Get(c.Request.Context(), artifactID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, artifact)
}

func (h *ArtifactHandler) ListByRelease(c *gin.Context) {
	releaseID, err := uuid.Parse(c.Param("releaseId"))
	if err != nil {
		writeError(c, err)
		return
	}
	artifacts, err := h.artifacts.ListByRelease(c.Request.Context(), releaseID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": artifacts})
}

func (h *ArtifactHandler) Update(c *gin.Context) {
	artifactID, err := uuid.Parse(c.Param("artifactId"))
	if err != nil {
		writeError(c, err)
		return
	}
	var input service.UpdateArtifactInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, err)
		return
	}
	artifact, err := h.artifacts.Update(c.Request.Context(), artifactID, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, artifact)
}

func (h *ArtifactHandler) ReplaceFile(c *gin.Context) {
	artifactID, err := uuid.Parse(c.Param("artifactId"))
	if err != nil {
		writeError(c, err)
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		writeError(c, err)
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		writeError(c, err)
		return
	}
	defer file.Close()

	artifact, err := h.artifacts.ReplaceFile(c.Request.Context(), artifactID, service.ReplaceArtifactFileInput{
		FileName: fileHeader.Filename,
		Body:     file,
	})
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, artifact)
}

func (h *ArtifactHandler) Archive(c *gin.Context) {
	artifactID, err := uuid.Parse(c.Param("artifactId"))
	if err != nil {
		writeError(c, err)
		return
	}
	if err := h.artifacts.Archive(c.Request.Context(), artifactID); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *ArtifactHandler) Download(c *gin.Context) {
	artifactID, err := uuid.Parse(c.Param("artifactId"))
	if err != nil {
		writeError(c, err)
		return
	}
	artifact, err := h.artifacts.Get(c.Request.Context(), artifactID)
	if err != nil {
		writeError(c, err)
		return
	}
	if h.stats != nil {
		_ = h.stats.RecordDownload(c.Request.Context(), artifact, service.RequestMeta{
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			ClientID:  c.Query("client_id"),
		})
	}
	c.Redirect(http.StatusTemporaryRedirect, artifact.FileURL)
}
