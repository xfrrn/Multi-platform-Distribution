package http

import (
	"net/http"

	"multi-platform-distribution/internal/auth"
	"multi-platform-distribution/internal/config"
	"multi-platform-distribution/internal/http/handlers"
	"multi-platform-distribution/internal/http/middleware"
	"multi-platform-distribution/internal/service"

	"github.com/gin-gonic/gin"
)

type Services struct {
	Apps      *service.AppService
	Releases  *service.ReleaseService
	Artifacts *service.ArtifactService
	Metadata  *service.MetadataService
	Stats     *service.StatsService
	Auth      *service.AuthService
	Tokens    *auth.TokenManager
}

func NewRouter(cfg config.Config, services Services) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	if cfg.StorageDriver == "local" {
		router.Static("/downloads", cfg.LocalStoragePath)
	}

	apps := handlers.NewAppHandler(services.Apps)
	releases := handlers.NewReleaseHandler(services.Releases)
	artifacts := handlers.NewArtifactHandler(services.Artifacts, services.Stats)
	latest := handlers.NewLatestHandler(services.Metadata, services.Stats)
	authHandler := handlers.NewAuthHandler(services.Auth)
	stats := handlers.NewStatsHandler(services.Stats)

	api := router.Group("/api")
	api.POST("/auth/login", authHandler.Login)
	api.GET("/latest/:appSlug/update.json", latest.GetJSON)
	api.GET("/latest/:appSlug/latest.yml", latest.GetElectronYAML)
	api.GET("/latest/:appSlug/appcast.xml", latest.GetAppcast)
	api.GET("/latest/:appSlug", latest.Get)
	api.GET("/artifacts/:artifactId/download", artifacts.Download)
	api.GET("/artifacts/:artifactId/download/:fileName", artifacts.Download)

	admin := api.Group("")
	admin.Use(middleware.AdminAuth(cfg.APIKey, services.Tokens))
	admin.GET("/config", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"anyshare_enabled": cfg.AnyshareEnabled})
	})
	admin.POST("/apps", apps.Create)
	admin.GET("/apps", apps.List)
	admin.GET("/stats/summary", stats.Summary)
	admin.GET("/stats/update-requests", stats.RecentUpdates)
	admin.GET("/stats/downloads", stats.RecentDownloads)
	admin.GET("/apps/:appId", apps.Get)
	admin.PATCH("/apps/:appId", apps.Update)
	admin.DELETE("/apps/:appId", apps.Archive)
	admin.GET("/apps/:appId/stats/summary", stats.AppSummary)
	admin.GET("/apps/:appId/stats/update-requests", stats.RecentUpdates)
	admin.GET("/apps/:appId/stats/downloads", stats.RecentDownloads)
	admin.POST("/apps/:appId/releases", releases.Create)
	admin.GET("/apps/:appId/releases", releases.ListByApp)
	admin.GET("/releases/:releaseId/stats", stats.ReleaseStats)
	admin.PATCH("/releases/:releaseId", releases.Update)
	admin.DELETE("/releases/:releaseId", releases.Archive)
	admin.GET("/releases/:releaseId/artifacts", artifacts.ListByRelease)
	admin.GET("/uploads/:uploadId", artifacts.UploadProgress)
	admin.GET("/artifacts/:artifactId", artifacts.Get)
	admin.PATCH("/artifacts/:artifactId", artifacts.Update)
	admin.PUT("/artifacts/:artifactId/file", artifacts.ReplaceFile)
	admin.DELETE("/artifacts/:artifactId", artifacts.Archive)
	admin.POST("/releases/:releaseId/artifacts", artifacts.Upload)

	return router
}
