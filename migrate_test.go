package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/rusq/wipemychat/internal/session"
)

func copyfile(t *testing.T, src, dst string) error {
	t.Helper()
	sf, err := os.Open(src)
	if err != nil {
		t.Fatalf("source: %s", err)
	}
	defer sf.Close()
	df, err := os.Create(dst)
	if err != nil {
		t.Fatalf("destination: %s", err)
	}
	defer df.Close()
	if _, err := io.Copy(df, sf); err != nil {
		t.Fatalf("copy: %s", err)
	}
	return nil
}

func Test_migratev120(t *testing.T) {
	type args struct {
		sessfile string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "migrates v1 file",
			args: args{sessfile: "testdata/testsessionv100.dat"},
			want: true,
		},
		{
			name: "doesn't touch v1.20 file",
			args: args{sessfile: "testdata/testsessionv120.dat"},
			want: false,
		},
		{
			name:    "invalid file",
			args:    args{sessfile: "testdata/invalidsession.dat"},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// substitute the source file with it's copy in a temporary directory
			tmpdir := t.TempDir()
			srcfilename := filepath.Base(tt.args.sessfile)
			dstfile := filepath.Join(tmpdir, srcfilename)
			copyfile(t, tt.args.sessfile, filepath.Join(tmpdir, srcfilename))
			migrated, err := migratev120(dstfile)
			if (err != nil) != tt.wantErr {
				t.Errorf("migratev120() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if migrated != tt.want {
				t.Errorf("migratev120() = %v, want %v", migrated, tt.want)
			}
			if migrated {
				// verify the file is valid v1.20
				sess := session.FileStorage{Path: dstfile}
				data, err := sess.LoadSession(context.Background())
				if err != nil {
					t.Errorf("migratev120() = %v, want %v", err, nil)
				}
				if data == nil {
					t.Errorf("migratev120() = %v, want %v", data, "not nil")
				}
				sigSz := len(v1signature)
				if !bytes.Equal(data[:sigSz], []byte(v1signature)) {
					t.Errorf("migratev120() = %v, want %v", data[:sigSz], v1signature)
				}
			}
		})
	}
}
