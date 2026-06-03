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
	artifacts := handlers.NewArtifactHandler(services.Artifacts)
	latest := handlers.NewLatestHandler(services.Metadata)
	authHandler := handlers.NewAuthHandler(services.Auth)

	api := router.Group("/api")
	api.POST("/auth/login", authHandler.Login)
	api.GET("/latest/:appSlug/update.json", latest.GetJSON)
	api.GET("/latest/:appSlug/latest.yml", latest.GetElectronYAML)
	api.GET("/latest/:appSlug/appcast.xml", latest.GetAppcast)
	api.GET("/latest/:appSlug", latest.Get)
	api.GET("/artifacts/:artifactId/download", artifacts.Download)

	admin := api.Group("")
	admin.Use(middleware.AdminAuth(cfg.APIKey, services.Tokens))
	admin.POST("/apps", apps.Create)
	admin.GET("/apps", apps.List)
	admin.GET("/apps/:appId", apps.Get)
	admin.POST("/apps/:appId/releases", releases.Create)
	admin.GET("/apps/:appId/releases", releases.ListByApp)
	admin.GET("/artifacts/:artifactId", artifacts.Get)
	admin.POST("/releases/:releaseId/artifacts", artifacts.Upload)

	return router
}
