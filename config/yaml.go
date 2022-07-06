package config

import (
	"bytes"
	"io/ioutil"
	"log"
	"path/filepath"
	"sort"

	"github.com/pkg/errors"
	"go.uber.org/config"
)

// MustLoadYAML mapper ini, error will fatal
func MustLoadYAML(path string, v interface{}) {
	if err := LoadYAML(path, v); err != nil {
		log.Fatalf("err: config file %s, %s", path, errors.Wrap(err, "load yaml"))
	}
}

// LoadYAML mapper ini
func LoadYAML(path string, v interface{}) error {
	files, err := filepath.Glob(path)
	if err != nil {
		return err
	}

	sort.Strings(files)

	if len(files) == 0 {
		return errors.New("at least one ini")
	}

	ls := make([]config.YAMLOption, 0, len(files))
	for _, file := range files {
		content, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}

		ls = append(ls, config.Source(bytes.NewReader(content)))
	}

	provider, err := config.NewYAML(ls...)
	if err != nil {
		return errors.Wrap(err, "merge yaml sources")
	}

	if err := provider.Get("").Populate(v); err != nil {
		return errors.Wrap(err, "mapper yaml")
	}

	return nil
}
