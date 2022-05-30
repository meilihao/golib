package virt

import (
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/go-version"
	"github.com/stretchr/testify/assert"
)

func TestGetOsinfos(t *testing.T) {
	ls, err := GetOsinfos()
	assert.Nil(t, err)
	assert.NotEmpty(t, ls)

	for _, v := range ls {
		spew.Dump(v)
		//fmt.Printf("os: %+v\n", v)
	}
}

func TestSemverCompare(t *testing.T) {
	v, _ := version.NewVersion("5.2")
	c, _ := version.NewConstraint(">= 5.0")
	fmt.Println(c.Check(v))
}

func TestGetOsinfosWithFilter(t *testing.T) {
	f := &OsinfoFilter{
		OsFamilies: []string{"linux", "winnt"},
	}

	ls, err := GetOsinfosWithFilter(f)
	assert.Nil(t, err)
	assert.NotEmpty(t, ls)

	for _, v := range ls {
		spew.Dump(v)
		//fmt.Printf("os: %+v\n", v)
	}
}
