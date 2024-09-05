package mysm4_test

import (
	mysm4 "bccrypto/encrypt_sm4"
	"bytes"
	"fmt"
	"testing"
)

// TestSm4: test encryption and decryption by sm4
func TestSm4(t *testing.T) {
	key := mysm4.GenerateKey()
	msg := []byte("0123456789abcdef012345678")
	encMsg := mysm4.Encrypt(key, msg)
	dec := mysm4.Decrypt(key, encMsg)

	if !bytes.Equal(msg, dec) {
		fmt.Println("sm4 self enc and dec failed", msg, dec, encMsg)
	} else {
		fmt.Println(key)
		fmt.Println(msg)
		fmt.Println(encMsg)
		fmt.Println(dec)
	}

}
