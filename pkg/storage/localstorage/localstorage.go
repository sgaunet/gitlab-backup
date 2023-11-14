package localstorage

import (
	"context"
	"io"
	"os"
)

type LocalStorage struct {
	dirpath string
}

func NewLocalStorage(dirpath string) *LocalStorage {
	return &LocalStorage{
		dirpath: dirpath,
	}
}

// SaveFile saves the file in localstorage
func (s *LocalStorage) SaveFile(ctx context.Context, archiveFilePath string, dstFilename string) error {
	src, err := os.Open(archiveFilePath)
	if err != nil {
		return err
	}
	defer src.Close()

	// save file in localstorage
	fDst, err := os.Create(s.dirpath + "/" + dstFilename)
	if err != nil {
		return err
	}
	defer fDst.Close()
	_, err = io.Copy(fDst, src)
	return err
}
