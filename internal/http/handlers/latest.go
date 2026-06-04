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
	stats    *service.StatsService
}

func NewLatestHandler(metadata *service.MetadataService, stats *service.StatsService) *LatestHandler {
	return &LatestHandler{metadata: metadata, stats: stats}
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
	query := service.LatestQuery{
		Channel:  c.Query("channel"),
		Platform: c.Query("platform"),
		Arch:     c.Query("arch"),
		ClientID: c.Query("client_id"),
	}
	result, err := h.metadata.LatestWithDetails(c.Request.Context(), c.Param("appSlug"), query)
	h.record(c, format, query, result, err)
	if err != nil {
		writeError(c, err)
		return
	}

	switch normalizeFormat(format) {
	case "yaml":
		body, err := metarender.RenderElectronLatest(result.Manifest)
		if err != nil {
			writeError(c, err)
			return
		}
		c.Data(http.StatusOK, "application/yaml; charset=utf-8", body)
	case "xml":
		body, err := metarender.RenderAppcast(c.Param("appSlug"), result.Manifest)
		if err != nil {
			writeError(c, err)
			return
		}
		c.Data(http.StatusOK, "application/xml; charset=utf-8", body)
	default:
		c.JSON(http.StatusOK, result.Manifest)
	}
}

func (h *LatestHandler) record(c *gin.Context, format string, query service.LatestQuery, result service.LatestResult, latestErr error) {
	if h.stats == nil {
		return
	}
	input := service.RecordUpdateInput{
		AppSlug: c.Param("appSlug"),
		Query:   query,
		Format:  normalizeFormat(format),
		Matched: latestErr == nil,
		Meta: service.RequestMeta{
			IP:        c.ClientIP(),
			UserAgent: c.Request.UserAgent(),
			ClientID:  c.Query("client_id"),
		},
	}
	if latestErr == nil {
		input.App = &result.App
		input.Release = &result.Release
		input.StagedHit = result.StagedHit
	}
	_ = h.stats.RecordUpdateRequest(c.Request.Context(), input)
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
