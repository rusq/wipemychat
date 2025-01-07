package session

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	sess "github.com/gotd/td/session"
	"github.com/rusq/encio"
)

// FileStorage implements SessionStorage for file system as file
// stored in Path.
type FileStorage struct {
	Path string
	mu   sync.Mutex
}

// LoadSession loads session from file.
func (f *FileStorage) LoadSession(_ context.Context) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	hFile, err := encio.Open(f.Path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, sess.ErrNotFound
		}
		return nil, fmt.Errorf("open: %w", err)
	}
	defer hFile.Close()

	data, err := io.ReadAll(hFile)
	if err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}

	return data, nil
}

// StoreSession stores session to file.
func (f *FileStorage) StoreSession(_ context.Context, data []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	hFile, err := encio.Create(f.Path)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	defer hFile.Close()

	_, err = io.Copy(hFile, bytes.NewReader(data))
	return err
}
