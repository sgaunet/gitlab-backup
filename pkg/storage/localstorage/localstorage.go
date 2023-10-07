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

func (s *LocalStorage) SaveFile(ctx context.Context, src io.Reader, dstFilename string, fileSize int64) error {
	// save file in localstorage
	fDst, err := os.Create(s.dirpath + "/" + dstFilename)
	if err != nil {
		return err
	}
	defer fDst.Close()
	_, err = io.Copy(fDst, src)
	return err
}
