package security

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	mrand "math/rand"
	"sync"
)

var (
	rsaList []*RSAPair
	rsaLen  int
	rsaLock = sync.RWMutex{}
)

type RSAPair struct {
	Public  *rsa.PublicKey
	Private *rsa.PrivateKey
	Index   int
}

func InitRSA(bits, num int) error {
	rsaLock.Lock()
	defer rsaLock.Unlock()

	if bits == 0 {
		bits = 4096
	}
	if bits < 2048 {
		bits = 2048
	}

	if num < 1 {
		num = 1
	}

	var err error
	var privateKey *rsa.PrivateKey
	rsaLen = num
	rsaList = make([]*RSAPair, num)

	for i := 0; i < num; i++ {
		privateKey, err = rsa.GenerateKey(rand.Reader, bits)
		if err != nil {
			return err
		}
		rsaList[i] = &RSAPair{
			Public:  &privateKey.PublicKey,
			Private: privateKey,
			Index:   i,
		}
	}

	return nil
}

func GetRSAByRandom() *RSAPair {
	rsaLock.RLock()
	defer rsaLock.RUnlock()

	return rsaList[mrand.Intn(rsaLen)]
}

func GetRSAByIndex(i int) *RSAPair {
	rsaLock.RLock()
	defer rsaLock.RUnlock()

	if i < 0 || i > rsaLen-1 {
		return nil
	}

	return rsaList[i]
}

func (p *RSAPair) Encrypt(data []byte) ([]byte, error) {
	encryptedBytes, err := rsa.EncryptOAEP(
		sha256.New(),
		rand.Reader,
		p.Public,
		data,
		nil)
	return encryptedBytes, err
}

func (p *RSAPair) Decrypt(data []byte) ([]byte, error) {
	decryptedBytes, err := p.Private.Decrypt(nil, data, &rsa.OAEPOptions{Hash: crypto.SHA256})
	return decryptedBytes, err
}
