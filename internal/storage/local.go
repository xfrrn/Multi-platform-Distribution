package storage

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type LocalStorage struct {
	root          string
	publicBaseURL string
}

func NewLocalStorage(root, publicBaseURL string) *LocalStorage {
	return &LocalStorage{
		root:          root,
		publicBaseURL: strings.TrimRight(publicBaseURL, "/"),
	}
}

func (s *LocalStorage) Save(ctx context.Context, key string, body io.Reader) (Object, error) {
	if err := ctx.Err(); err != nil {
		return Object{}, err
	}

	cleanKey := filepath.Clean(filepath.FromSlash(key))
	target := filepath.Join(s.root, cleanKey)
	rootAbs, err := filepath.Abs(s.root)
	if err != nil {
		return Object{}, err
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return Object{}, err
	}
	if targetAbs != rootAbs && !strings.HasPrefix(targetAbs, rootAbs+string(filepath.Separator)) {
		return Object{}, errors.New("storage key escapes local storage root")
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return Object{}, err
	}

	file, err := os.Create(target)
	if err != nil {
		return Object{}, err
	}
	defer file.Close()

	hash := sha512.New()
	size, err := io.Copy(io.MultiWriter(file, hash), body)
	if err != nil {
		return Object{}, err
	}

	return Object{
		Key:    filepath.ToSlash(cleanKey),
		URL:    s.publicBaseURL + "/downloads/" + filepath.ToSlash(cleanKey),
		Size:   size,
		SHA512: base64.StdEncoding.EncodeToString(hash.Sum(nil)),
	}, nil
}

func (s *LocalStorage) PublicPath() string {
	return s.root
}
