package ssh

import (
	"testing"
)

func TestPassword(t *testing.T) {
	conf := &ClientConfig{
		Host:         "localhost",
		User:         "chen",
		Password:     "xxx",
		DisableAgent: true,
	}

	c, err := NewClient(conf)
	if err != nil {
		t.Fatal(err)
	}

	r, err := c.Execute("echo 1")
	if err != nil {
		t.Fatal(err)
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

	r, err := c.Execute("ls ~")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(r.String())
}
