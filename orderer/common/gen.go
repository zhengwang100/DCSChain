package common

import (
	crand "crypto/rand"
	"encoding/base64"
	"math/rand"
	"time"
)

// GenerateSecureRandomStringSlice: randomly generate a secure string slice
// params:
// - reqNum: the amount of string(request) to be generated
// - length: the length of each string
// return:
// - a slice of generated random strings
func GenerateSecureRandomStringSlice(reqNum int, length int) []string {
	res := make([]string, reqNum)
	for i := range res {
		res[i] = GenerateSecureRandomString(length)
	}
	return res
}

// GenerateSecureRandom2ByteSlice: randomly generate multiple 2D random byte slices
// params:
// - count: the amount of []byte ([]byte format of the request) to be generated
// - length: the length of each []byte
// return:
// - a slice of generated random []byte
func GenerateSecureRandom2ByteSlice(count int, length int) [][]byte {
	res := make([][]byte, count)
	for i := 0; i < count; i++ {
		res[i] = GenerateSecureRandomByteSlice(length)
	}
	return res
}

// GenerateSecureRandom2ByteSlice: randomly generate a random byte slice
// params:
// - length: the length of each []byte
// return:
// - a slice of generated random []byte
func GenerateSecureRandomByteSlice(length int) []byte {
	str := GenerateSecureRandomString(length)
	return []byte(str)
}

// GenerateRandomString: randomly generate string
// params:
// - length: the length of each string
// return:
// - a random strings
// note:!!!!!!! the GenerateSecureRandomString function is recommended for security!!!!!!!
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	defaultRand := rand.New(rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[defaultRand.Intn(len(charset))]
	}
	return string(b)
}

// GenerateSecureRandomString: randomly generate secure string
// params:
// - length: the length of each string
// return:
// - a random strings
func GenerateSecureRandomString(length int) string {
	randomBytes := make([]byte, length)
	_, err := crand.Read(randomBytes)
	if err != nil {
		return ""
	}

	// base64 encoding is used to ensure printable strings are generated
	randomString := base64.URLEncoding.EncodeToString(randomBytes)[:length]
	return randomString
}
