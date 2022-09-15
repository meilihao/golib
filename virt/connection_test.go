package virt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnectionManager(t *testing.T) {
	m, err := NewConnectionManager(5, 5)
	assert.Nil(t, err)

	host := "127.0.0.1"

	conn, err := m.GetConnection(host, "", "", LibvirtUriTypeSocket)
	assert.Nil(t, err)
	assert.NotNil(t, conn)

	conn, err = m.GetConnection(host, "", "", LibvirtUriTypeSocket)
	assert.Nil(t, err)
	assert.NotNil(t, conn)
}
