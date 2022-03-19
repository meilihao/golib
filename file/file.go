package file

import (
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
	f, err := os.Stat(fp)
	if errors.Is(err, fs.ErrNotExist) {
		return false
	}
	return true
}
