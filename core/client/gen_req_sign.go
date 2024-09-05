package client

import (
	"bufio"
	common "common"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"ssm2"
	"strconv"

	"github.com/xlcetc/cryptogm/sm/sm2"
)

// GenNewReqs: generate the new request and signature, store it to file
// params:
// - keyPath: the key path
// - count: the number of requests to be generated
// - length: the length of request
func GenNewReqs(keyPath string, count int, length int) {
	sk := ssm2.ReadKey(keyPath + "private.pem")
	pk := ssm2.ReadKey(keyPath + "public.pem")
	// length := 128

	if sk == nil || pk == nil {
		fmt.Println("Key read error")
		return
	}

	file, err := os.Create("../../config/request/request" + strconv.Itoa(length))
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	// 创建 bufio.Writer
	writer := bufio.NewWriter(file)

	// sign for the random request
	for i := 0; i < count; i++ {

		// get a secure random byte silce which is request in fact
		req := common.GenerateSecureRandomByteSlice(length)

		// sign for the random request
		sign, err := sm2.Sm2Sign(sk, pk, req)
		if err != nil {
			fmt.Println("Error sign", err)
			i--
			continue
		}

		// write request to file and wrap the line
		req, _ = json.Marshal(req)
		_, err = writer.Write(req)
		if err != nil {
			log.Fatalf("Failed to write requests to file: %v", err)
			i--
			continue
		}
		_, err = writer.WriteString("\n")
		if err != nil {
			log.Fatalf("Failed to write newline to file: %v", err)
		}

		// write signature to file and wrap the line
		sign, _ = json.Marshal(sign)
		_, err = writer.Write(sign)
		if err != nil {
			log.Fatalf("Failed to write requests to file: %v", err)
			i--
			continue
		}
		_, err = writer.WriteString("\n")
		if err != nil {
			log.Fatalf("Failed to write newline to file: %v", err)
		}

		// refresh buffer
		err = writer.Flush()
		if err != nil {
			log.Fatalf("Failed to flush writer: %v", err)
		}
	}
}
