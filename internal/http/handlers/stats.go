package handlers

import (
	"net/http"
	"strconv"
	"time"

	"multi-platform-distribution/internal/domain"
	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type StatsHandler struct {
	stats *service.StatsService
}

func NewStatsHandler(stats *service.StatsService) *StatsHandler {
	return &StatsHandler{stats: stats}
}

func (h *StatsHandler) Summary(c *gin.Context) {
	filter, err := statsFilter(c, nil, nil)
	if err != nil {
		writeError(c, err)
		return
	}
	summary, err := h.stats.Summary(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *StatsHandler) AppSummary(c *gin.Context) {
	appID, err := uuid.Parse(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	filter, err := statsFilter(c, &appID, nil)
	if err != nil {
		writeError(c, err)
		return
	}
	summary, err := h.stats.Summary(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *StatsHandler) ReleaseStats(c *gin.Context) {
	releaseID, err := uuid.Parse(c.Param("releaseId"))
	if err != nil {
		writeError(c, err)
		return
	}
	filter, err := statsFilter(c, nil, &releaseID)
	if err != nil {
		writeError(c, err)
		return
	}
	summary, err := h.stats.Summary(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, summary)
}

func (h *StatsHandler) RecentUpdates(c *gin.Context) {
	appID, err := optionalAppID(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	filter, err := statsFilter(c, appID, nil)
	if err != nil {
		writeError(c, err)
		return
	}
	events, err := h.stats.RecentUpdates(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": events})
}

func (h *StatsHandler) RecentDownloads(c *gin.Context) {
	appID, err := optionalAppID(c.Param("appId"))
	if err != nil {
		writeError(c, err)
		return
	}
	filter, err := statsFilter(c, appID, nil)
	if err != nil {
		writeError(c, err)
		return
	}
	events, err := h.stats.RecentDownloads(c.Request.Context(), filter)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": events})
}

func optionalAppID(value string) (*uuid.UUID, error) {
	if value == "" {
		return nil, nil
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func queryLimit(c *gin.Context) int {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		return 50
	}
	return limit
}

func queryOffset(c *gin.Context) int {
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		return 0
	}
	return offset
}

func statsFilter(c *gin.Context, appID, releaseID *uuid.UUID) (domain.StatsFilter, error) {
	dateFrom, err := parseStatsTime(c.Query("date_from"), false)
	if err != nil {
		return domain.StatsFilter{}, err
	}
	dateTo, err := parseStatsTime(c.Query("date_to"), true)
	if err != nil {
		return domain.StatsFilter{}, err
	}
	return domain.StatsFilter{
		AppID:     appID,
		ReleaseID: releaseID,
		DateFrom:  dateFrom,
		DateTo:    dateTo,
		Channel:   c.Query("channel"),
		Platform:  c.Query("platform"),
		Arch:      c.Query("arch"),
		ClientID:  c.Query("client_id"),
		Limit:     queryLimit(c),
		Offset:    queryOffset(c),
	}, nil
}

func parseStatsTime(value string, endOfDay bool) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, err
	}
	if endOfDay {
		parsed = parsed.Add(24*time.Hour - time.Nanosecond)
	}
	return &parsed, nil
}
