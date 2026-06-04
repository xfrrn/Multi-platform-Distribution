package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"multi-platform-distribution/internal/anyshare"
	"multi-platform-distribution/internal/auth"
	"multi-platform-distribution/internal/config"
	"multi-platform-distribution/internal/database"
	httpapi "multi-platform-distribution/internal/http"
	"multi-platform-distribution/internal/repository/postgres"
	"multi-platform-distribution/internal/service"
	"multi-platform-distribution/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

type App struct {
	cfg    config.Config
	db     *pgxpool.Pool
	server *http.Server
}

func New(cfg config.Config) (*App, error) {
	if cfg.DatabaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}
	if err := db.Ping(context.Background()); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}
	if cfg.AutoMigrate {
		if err := database.Migrate(context.Background(), db); err != nil {
			db.Close()
			return nil, fmt.Errorf("migrate database: %w", err)
		}
	}

	var objectStorage service.ObjectStorage
	switch cfg.StorageDriver {
	case "local", "":
		objectStorage = storage.NewLocalStorage(cfg.LocalStoragePath, cfg.PublicBaseURL)
	case "s3", "minio", "oss":
		objectStorage, err = storage.NewS3Storage(context.Background(), storage.S3Config{
			Endpoint:      cfg.S3Endpoint,
			Region:        cfg.S3Region,
			Bucket:        cfg.S3Bucket,
			AccessKey:     cfg.S3AccessKey,
			SecretKey:     cfg.S3SecretKey,
			PublicBaseURL: cfg.S3PublicBaseURL,
			UseSSL:        cfg.S3UseSSL,
		})
		if err != nil {
			db.Close()
			return nil, err
		}
	default:
		db.Close()
		return nil, fmt.Errorf("unsupported STORAGE_DRIVER %q", cfg.StorageDriver)
	}

	var anyshareClient *anyshare.Client
	if cfg.AnyshareEnabled {
		anyshareClient, err = anyshare.NewClient(context.Background(), anyshare.Config{
			BaseURL:     cfg.AnyshareBaseURL,
			SharingLink: cfg.AnyshareShareLink,
			UploadPath:  cfg.AnyshareUploadPath,
			Timeout:     cfg.AnyshareTimeout,
		})
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("configure anyshare: %w", err)
		}
	}

	repo := postgres.New(db)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTTTL)
	apps := service.NewAppService(repo)
	releases := service.NewReleaseService(repo)
	artifacts := service.NewArtifactServiceWithAnyshare(repo, objectStorage, anyshareClient, cfg.PublicBaseURL)
	metadata := service.NewMetadataService(repo)
	stats := service.NewStatsService(repo)
	authService := service.NewAuthService(repo, tokenManager)
	if err := authService.BootstrapDefaultAdmin(context.Background(), service.BootstrapAdminInput{
		Email:    cfg.BootstrapEmail,
		Name:     cfg.BootstrapName,
		Password: cfg.BootstrapPassword,
	}); err != nil {
		db.Close()
		return nil, fmt.Errorf("bootstrap admin: %w", err)
	}

	router := httpapi.NewRouter(cfg, httpapi.Services{
		Apps:      apps,
		Releases:  releases,
		Artifacts: artifacts,
		Metadata:  metadata,
		Stats:     stats,
		Auth:      authService,
		Tokens:    tokenManager,
	})

	return &App{
		cfg: cfg,
		db:  db,
		server: &http.Server{
			Addr:    cfg.HTTPAddr,
			Handler: router,
		},
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- a.server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (a *App) Shutdown(ctx context.Context) error {
	var err error
	if a.server == nil {
		if a.db != nil {
			a.db.Close()
		}
		return err
	}
	err = a.server.Shutdown(ctx)
	if a.db != nil {
		a.db.Close()
	}
	return err
}
