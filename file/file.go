package file

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
)

func IsFile(fp string) bool {
	f, err := os.Stat(fp)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return !f.IsDir()
}

func IsDir(fp string) bool {
	f, err := os.Stat(fp)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return f.IsDir()
}

func IsExist(fp string) bool {
	_, err := os.Stat(fp)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}

func FileValue(p string) string {
	data, _ := os.ReadFile(p)
	return string(bytes.TrimSpace(data))
}
