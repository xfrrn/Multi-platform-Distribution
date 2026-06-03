package handlers

import (
	"net/http"
	"strings"

	metarender "multi-platform-distribution/internal/metadata"
	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
)

type LatestHandler struct {
	metadata *service.MetadataService
}

func NewLatestHandler(metadata *service.MetadataService) *LatestHandler {
	return &LatestHandler{metadata: metadata}
}

func (h *LatestHandler) Get(c *gin.Context) {
	h.render(c, c.Query("format"))
}

func (h *LatestHandler) GetJSON(c *gin.Context) {
	h.render(c, "json")
}

func (h *LatestHandler) GetElectronYAML(c *gin.Context) {
	h.render(c, "latest.yml")
}

func (h *LatestHandler) GetAppcast(c *gin.Context) {
	h.render(c, "appcast.xml")
}

func (h *LatestHandler) render(c *gin.Context, format string) {
	manifest, err := h.metadata.Latest(c.Request.Context(), c.Param("appSlug"), service.LatestQuery{
		Channel:  c.Query("channel"),
		Platform: c.Query("platform"),
		Arch:     c.Query("arch"),
	})
	if err != nil {
		writeError(c, err)
		return
	}

	switch normalizeFormat(format) {
	case "yaml":
		body, err := metarender.RenderElectronLatest(manifest)
		if err != nil {
			writeError(c, err)
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", body)
	case "xml":
		body, err := metarender.RenderAppcast(c.Param("appSlug"), manifest)
		if err != nil {
			writeError(c, err)
			return
		}
		c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
	default:
		c.JSON(http.StatusOK, manifest)
	}
}

func normalizeFormat(format string) string {
	format = strings.ToLower(strings.TrimSpace(format))
	switch format {
	case "yml", "yaml", "latest.yml", "electron":
		return "yaml"
	case "xml", "appcast", "appcast.xml":
		return "xml"
	default:
		return "json"
	}
}
