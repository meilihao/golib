package ssh

import (
	"testing"
)

func TestPassword(t *testing.T) {
	conf := &ClientConfig{
		Host:         "47.111.1.49",
		User:         "xxx",
		Password:     "xxx",
		DisableAgent: true,
	}

	c, err := NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	r := c.Execute("ls ~")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	t.Log(r.String())
}

func TestEd25519(t *testing.T) {
	conf := &ClientConfig{
		Host:         "47.111.1.49",
		User:         "chen",
		PrivateKey:   "/home/chen/.ssh/aliyun_ed25519",
		DisableAgent: true,
		Passphrase:   "xxx",
	}

	c, err := NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	r := c.Execute("ls ~")
	if r.Error != nil {
		t.Fatal(r.Error)
	}
	t.Log(r.String())
}
