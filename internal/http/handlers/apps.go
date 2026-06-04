package handlers

import (
	"net/http"

	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AppHandler struct {
	apps *service.AppService
}

func NewAppHandler(apps *service.AppService) *AppHandler {
	return &AppHandler{apps: apps}
}

func (h *AppHandler) Create(c *gin.Context) {
	var input service.CreateAppInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, err)
		return
	}
	app, err := h.apps.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, app)
}

func (h *AppHandler) List(c *gin.Context) {
	apps, err := h.apps.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": apps})
}

func (h *AppHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	app, err := h.apps.Get(c.Request.Context(), id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func (h *AppHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	var input service.UpdateAppInput
	if err := c.ShouldBindJSON(&input); err != nil {
		writeError(c, err)
		return
	}
	app, err := h.apps.Update(c.Request.Context(), id, input)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, app)
}

func (h *AppHandler) Archive(c *gin.Context) {
	id, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	if err := h.apps.Archive(c.Request.Context(), id); err != nil {
		writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
