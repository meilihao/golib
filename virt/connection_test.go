package virt

import (
	"testing"
	"time"

	"github.com/meilihao/golib/v2/cmd"
	"github.com/stretchr/testify/assert"
)

func TestConnectionManager(t *testing.T) {
	m, err := NewConnectionManager(5, 5)
	assert.Nil(t, err)

	host := "127.0.0.1"

	conn, err := m.GetConnection(host, "", "", LibvirtUriTypeSocket)
	assert.Nil(t, err)
	assert.NotNil(t, conn)

	go func() {
		time.Sleep(time.Second)
		_, err := cmd.CmdCombinedBash(nil, "systemctl restart libvirtd.service")
		assert.Nil(t, err)
	}()
	time.Sleep(10 * time.Second)

	conn, err = m.GetConnection(host, "", "", LibvirtUriTypeSocket)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}
