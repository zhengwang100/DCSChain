package common_test

import (
	"common"
	"testing"
)

// TestGenerateSecureRandomByteSlice: test the stability of the function
func TestGenerateSecureRandomByteSlice(t *testing.T) {
	count := 1000000
	for i := 0; i < count; i++ {
		common.GenerateSecureRandomByteSlice(100)
	}
}
