package handlers

import (
	"net/http"

	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReleaseHandler struct {
	releases *service.ReleaseService
}

func NewReleaseHandler(releases *service.ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{releases: releases}
}

func (h *ReleaseHandler) Create(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	var input service.CreateReleaseInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, err)
		return
	}
	input.AppID = appID
	release, err := h.releases.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, release)
}

func (h *ReleaseHandler) ListByApp(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	releases, err := h.releases.ListByApp(c.Request.Context(), appID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": releases})
}
