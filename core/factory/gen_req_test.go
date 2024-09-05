package factory_test

import (
	"factory"
	"fmt"
	"ssm2"
	"testing"
	"time"

	"github.com/xlcetc/cryptogm/sm/sm2"
)

// TestReadReq: read the request and verify the signature
func TestReadReq(t *testing.T) {
	pk := ssm2.ReadKey("../../config/client/public.pem")
	length := 128
	count := 1000

	req := factory.ReadReq(count, length)
	fmt.Println(len(req))
	start := time.Now()
	for j := 0; j < count; j++ {
		for i := 0; i < count; i++ {
			if !sm2.Sm2Verify(req[i].Sign, pk, req[i].Cmd) {
				fmt.Println("error", i)
				return
			}
		}
	}
	fmt.Println("all true", time.Since(start)/time.Duration(count*1000))
}
