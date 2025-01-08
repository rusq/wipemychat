package main

import (
	"bytes"
	"context"
	"fmt"
	"os"

	tds "github.com/gotd/td/session"

	"github.com/rusq/wipemychat/internal/session"
)

const v1signature = `{"Version":1`

// migratev120 migrates session file from v1 to v1.2.0+ (enables encryption).
// sessfile is the path to the session file. It returns true if the file was
// migrated, false if it was already migrated or invalid.
func migratev120(sessfile string) (bool, error) {
	f, err := os.Open(sessfile)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	defer f.Close()
	if f, err := f.Stat(); err != nil {
		return false, err
	} else if f.Size() == 0 {
		return false, nil
	}
	b := make([]byte, len(v1signature))
	if n, err := f.Read(b); err != nil {
		return false, fmt.Errorf("failed to read session file: %w", err)
	} else if n != len(v1signature) {
		return false, fmt.Errorf("invalid session file")
	}

	if !bytes.Equal(b[:], []byte(v1signature)) {
		// already migrated or invalid
		return false, nil
	}
	// needs to be migrated
	if err := f.Close(); err != nil {
		return false, fmt.Errorf("close error: %w", err)
	}
	v1loader := tds.FileStorage{Path: sessfile}
	sess, err := v1loader.LoadSession(context.Background())
	if err != nil {
		return false, fmt.Errorf("failed to load session: %w", err)
	}

	// overwrite with new version
	v120loader := session.FileStorage{Path: sessfile}
	if err := v120loader.StoreSession(context.Background(), sess); err != nil {
		return false, fmt.Errorf("failed to save session: %w", err)
	}
	return true, nil
}
