package helpers

import (
	"os"
	"testing"
	"time"
)

func TestFakeFileInfo(t *testing.T) {
	tests := []struct {
		mode     os.FileMode
		expIsDir bool
	}{
		{0777, false},
		{0777 | os.ModeDir, true},
	}

	for _, tc := range tests {
		name := "name"
		size := int64(123)
		modTime := time.Now()
		info := NewFakeFileInfo(name, size, tc.mode, modTime)

		if info.Name() != name {
			t.Errorf("got name %v, want %v", info.Name(), name)
		}
		if info.Size() != size {
			t.Errorf("got size %v, want %v", info.Size(), size)
		}
		if info.Mode() != tc.mode {
			t.Errorf("got mode %v, want %v", info.Mode(), tc.mode)
		}
		if info.ModTime() != modTime {
			t.Errorf("got modTime %v, want %v", info.ModTime(), modTime)
		}
		if info.IsDir() != tc.expIsDir {
			t.Errorf("got isDir %v, want %v", info.IsDir(), tc.expIsDir)
		}
		if info.Sys() != nil {
			t.Errorf("got sys %v, want nil", info.Sys())
		}
	}

}
