package config

import (
	"log"
	"path/filepath"
	"sort"

	"github.com/meilihao/golib/v1/convert"

	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

// MustLoadINI mapper ini, error will fatal
func MustLoadINI(path string, v interface{}) {
	if err := LoadINI(path, v); err != nil {
		log.Fatalf("err: config file %s, %s", path, errors.Wrap(err, "load ini"))
	}
}

// LoadINI mapper ini
func LoadINI(path string, v interface{}) error {
	files, err := filepath.Glob(path)
	if err != nil {
		return err
	}

	sort.Strings(files)

	if len(files) == 0 {
		return errors.New("at least one ini")
	}

	ls := convert.Strings2Interfaces(files)
	cfg, err := ini.LoadSources(ini.LoadOptions{}, ls[0], ls[1:]...)
	if err != nil {
		return errors.Wrap(err, "load ini sources")
	}

	if err = cfg.MapTo(v); err != nil {
		return errors.Wrap(err, "mapper ini")
	}

	return nil
}
