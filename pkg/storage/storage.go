package storage

import (
	"context"
)

type Storage interface {
	SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) error
}
