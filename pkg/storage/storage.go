package storage

import (
	"context"
	"io"
)

type Storage interface {
	SaveFile(ctx context.Context, src io.Reader, dstFilename string, fileSize int64) error
}
