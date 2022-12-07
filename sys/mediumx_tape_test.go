package sys

import (
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/meilihao/golib/v2/cmd"
	"github.com/stretchr/testify/assert"
)

func TestTapeByIdPaths(t *testing.T) {
	byIds, err := TapeByIdPaths()
	assert.Nil(t, err)

	raw, err := cmd.CmdCombinedBash(nil, "lsscsi -g")
	assert.Nil(t, err)

	if strings.Contains(string(raw), "mediumx") {
		assert.NotEmpty(t, byIds)

		spew.Dump(byIds)
	}
}

func TestGetMediumxs(t *testing.T) {
	ls, err := GetMediumxs()
	assert.Nil(t, err)

	raw, err := cmd.CmdCombinedBash(nil, "lsscsi -g")
	assert.Nil(t, err)

	if strings.Contains(string(raw), "mediumx") {
		assert.NotEmpty(t, ls)

		spew.Dump(ls)
	}
}
