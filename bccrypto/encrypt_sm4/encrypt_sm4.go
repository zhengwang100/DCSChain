package mysm4

import (
	"crypto/rand"

	"github.com/xlcetc/cryptogm/sm/sm4"
)

// Encrypt: encrypt the message by the key offered
// params:
// -key: the key of sm4
// -msg: the message need to be encrypted
// return encrypted message
func Encrypt(key []byte, msg []byte) []byte {
	c, err := sm4.Sm4Ecb(key, msg, sm4.ENC)
	if err == nil {
		return c
	}
	return nil
}

// Decrypt: Decrypt the encrypted message by the key offered
// params:
// -key: the key of sm4
// -msg: the encrypted message need to be decrypted
// return decrypted message
func Decrypt(key []byte, msg []byte) []byte {
	d, err := sm4.Sm4Ecb(key, msg, sm4.DEC)
	if err == nil {
		return d
	}
	return nil
}

// GenerateKey: generate sm4 key
func GenerateKey() []byte {
	keyLength := 16
	key := make([]byte, keyLength)
	// rand.Seed(time.Now().UnixNano())

	_, err := rand.Read(key)
	if err != nil {
		return nil
	}

	return key
}
