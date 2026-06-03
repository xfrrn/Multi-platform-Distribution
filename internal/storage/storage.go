package storage

import (
	"context"
	"io"
)

type Object struct {
	Key    string
	URL    string
	Size   int64
	SHA512 string
}

type Storage interface {
	Save(ctx context.Context, key string, body io.Reader) (Object, error)
	PublicPath() string
}
