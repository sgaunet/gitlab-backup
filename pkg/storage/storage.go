// Package storage defines the interface for backup storage backends.
package storage

import (
	"context"
)

// Storage interface defines methods for saving backup files.
type Storage interface {
	SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) error
}
