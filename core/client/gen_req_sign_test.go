package client_test

import (
	"deltachain/core/client"
	"testing"
)

// TestGenNewReqs: test the function of generating new requests
func TestGenNewReqs(t *testing.T) {
	count := 10
	lenght := 50
	path := "../../config/client/"
	client.GenNewReqs(path, count, lenght)
}
