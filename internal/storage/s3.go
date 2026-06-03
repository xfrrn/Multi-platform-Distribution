package storage

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Config struct {
	Endpoint      string
	Region        string
	Bucket        string
	AccessKey     string
	SecretKey     string
	PublicBaseURL string
	UseSSL        bool
}

type S3Storage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL string
	endpoint      string
	useSSL        bool
}

func NewS3Storage(ctx context.Context, cfg S3Config) (*S3Storage, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, errors.New("S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY, and S3_SECRET_KEY are required")
	}

	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("create s3 client: %w", err)
	}

	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("check bucket: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{Region: cfg.Region}); err != nil {
			return nil, fmt.Errorf("create bucket: %w", err)
		}
	}

	return &S3Storage{
		client:        client,
		bucket:        cfg.Bucket,
		publicBaseURL: strings.TrimRight(cfg.PublicBaseURL, "/"),
		endpoint:      strings.TrimRight(cfg.Endpoint, "/"),
		useSSL:        cfg.UseSSL,
	}, nil
}

func (s *S3Storage) Save(ctx context.Context, key string, body io.Reader) (Object, error) {
	temp, err := os.CreateTemp("", "mpd-upload-*")
	if err != nil {
		return Object{}, err
	}
	defer os.Remove(temp.Name())
	defer temp.Close()

	hash := sha512.New()
	size, err := io.Copy(io.MultiWriter(temp, hash), body)
	if err != nil {
		return Object{}, err
	}
	if _, err := temp.Seek(0, io.SeekStart); err != nil {
		return Object{}, err
	}

	_, err = s.client.PutObject(ctx, s.bucket, key, temp, size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	})
	if err != nil {
		return Object{}, fmt.Errorf("upload object: %w", err)
	}

	return Object{
		Key:    key,
		URL:    s.publicURL(key),
		Size:   size,
		SHA512: base64.StdEncoding.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (s *S3Storage) PublicPath() string {
	return s.publicBaseURL
}

func (s *S3Storage) publicURL(key string) string {
	escapedKey := escapeObjectKey(key)
	if s.publicBaseURL != "" {
		return s.publicBaseURL + "/" + escapedKey
	}

	scheme := "http"
	if s.useSSL {
		scheme = "https"
	}
	return scheme + "://" + s.endpoint + "/" + url.PathEscape(s.bucket) + "/" + escapedKey
}

func escapeObjectKey(key string) string {
	parts := strings.Split(key, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
