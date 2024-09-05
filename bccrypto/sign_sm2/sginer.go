package ssm2

import (
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/xlcetc/cryptogm/sm/sm2"
)

// Signer: the unit for sm2 signature
type Signer struct {
	ID  string            // signer id, unique identification of the signer
	Pk  []byte            // public key
	Sk  []byte            // secret key
	Pks map[string][]byte // a map with other <id, pk>
}

// NewSigners: generate new signers
// params:
// -num: the number of signers that need to be generated
// return slice of new signers
func NewSigners(num int) []*Signer {
	rand := rand.Reader
	newSigners := make([]*Signer, num)
	for i := 0; i < num; i++ {
		sk, pk, err := sm2.Sm2KeyGen(rand)
		if err != nil {
			i--
			fmt.Println(err)
			continue
		}
		newSigners[i] = &Signer{
			ID:  "r_" + strconv.Itoa(i),
			Pk:  pk,
			Sk:  sk,
			Pks: make(map[string][]byte),
		}
	}

	// add other signers' PK
	for i := 0; i < num; i++ {
		for j := 0; j < num; j++ {
			newSigners[j].Pks[newSigners[i].ID] = newSigners[i].Pk
		}
	}
	return newSigners
}

// Sign: sgin for message
// params:
// -msg: the message that need to be signed
// return signature
func (s *Signer) Sign(msg []byte) []byte {
	sign, err := sm2.Sm2Sign(s.Sk, s.Pk, msg)
	if err != nil {

		fmt.Println(err)
		return nil
	}
	return sign
}

// VerifySign: verify that the signature is correct
// params:
// -msg: the signature that need to be verified
// return true or false
func (s *Signer) VerifySign(signerName string, sign []byte, msg []byte) bool {
	if pk, ok := s.Pks[signerName]; ok {
		return sm2.Sm2Verify(sign, pk, msg)
	} else {
		fmt.Println("VerifySign Error : not public key", signerName)
		return false
	}
}

// StorePK: store public key to file
func (s *Signer) StorePK() {
	WriteKey(s.Pk, "E:/MyOwnDoc/Project/GoProject/src/DeltaChain/config/public.pem")
}

// GetPKFromFile: get public key from file
func (s *Signer) GetPKFromFile() []byte {
	return ReadKey("E:/MyOwnDoc/Project/GoProject/src/DeltaChain/config/client/public.pem")
}

// StorePK: store private key(SK) to file
func (s *Signer) StoreSK() {
	WriteKey(s.Pk, "E:/MyOwnDoc/Project/GoProject/src/DeltaChain/config/client/private.pem")
}

// GetPKFromFile: get private key(SK) from file
func (s *Signer) GetSKFromFile() []byte {
	return ReadKey("E:/MyOwnDoc/Project/GoProject/src/DeltaChain/config/client/private.pem")
	// return ReadKey("../public.pem")
}

// WriteKey: wirte the key to path
func WriteKey(k []byte, path string) bool {
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: k,
	}
	publicKeyPEMBytes := pem.EncodeToMemory(publicKeyPEM)

	if err := os.WriteFile(path, publicKeyPEMBytes, 0644); err != nil {
		log.Fatalf("Failed to write public key to file: %v", err)
		return false
	}
	return true
}

// ReadKey: read key from path
func ReadKey(path string) []byte {
	publicKeyPEMBytesFromFile, err := os.ReadFile(path)
	if err != nil {
		// fmt.Printf("Failed to read key from file: %v\n", err)
		return nil
	}
	pk, _ := pem.Decode(publicKeyPEMBytesFromFile)
	return pk.Bytes
}
