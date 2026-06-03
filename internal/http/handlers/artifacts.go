package handlers

import (
	"net/http"

	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ArtifactHandler struct {
	artifacts *service.ArtifactService
}

func NewArtifactHandler(artifacts *service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{artifacts: artifacts}
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
	c.Redirect(http.StatusTemporaryRedirect, artifact.FileURL)
}
