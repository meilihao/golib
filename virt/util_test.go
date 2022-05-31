package virt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiskFromNumber(t *testing.T) {
	g := NewDiskFromNumber("sata", 1)
	v1 := g.Generate()
	v2 := g.Generate()
	assert.Equal(t, v1, "sda")
	assert.Equal(t, v2, "sdb")

	g = NewDiskFromNumber("sata", 10)
	v1 = g.Generate()
	assert.Equal(t, v1, "sdj")

	g = NewDiskFromNumber("sata", 26)
	v1 = g.Generate()
	assert.Equal(t, v1, "sdz")

	g = NewDiskFromNumber("sata", 100)
	v1 = g.Generate()
	assert.Equal(t, v1, "sdcv")

	g = NewDiskFromNumberByName("sdca")
	v1 = g.Generate()
	assert.Equal(t, v1, "sdcb")
}
