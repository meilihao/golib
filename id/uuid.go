package id

import (
	"strings"

	"github.com/google/uuid"
)

func NewUUIDV4(needReplace bool) string {
	if needReplace {
		return strings.ReplaceAll(uuid.NewString(), "-", "")
	}

	return uuid.NewString()
}

