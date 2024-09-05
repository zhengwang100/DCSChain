package client_test

import (
	"bufio"
	common "common"
	"crypto/rand"
	"deltachain/core/client"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/xlcetc/cryptogm/sm/sm2"
)

// TestClient: test the client
func TestClient(t *testing.T) {
	c := client.NewClient("127.0.0.1", "20000")
	count := 1000000
	st := time.Now()
	for i := 0; i < count; i++ {
		if c.Mu.TryLock() {
			c.Mu.Unlock()
		}
	}
	fmt.Println(time.Since(st) / time.Duration(count))
	// fmt.Println(c)
}

// TestRW: test the time of read and wirte the request
func TestRW(t *testing.T) {
	count := 10000
	length := 100
	filename := "requests.json"

	sk, pk, err := sm2.Sm2KeyGen(rand.Reader)
	// reqs := make([][]byte, count)
	// signs := make([][]byte, count)
	if err != nil {
		return
	}

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("Failed to create file: %v", err)
	}

	wst := time.Now().UnixNano() / 1e6
	// create bufio.Writer
	writer := bufio.NewWriter(file)
	for i := 0; i < count; i++ {
		req := common.GenerateSecureRandomByteSlice(length)
		// dig := sm3.SumSM3(req)
		sign, err := sm2.Sm2Sign(sk, pk, req)
		if err != nil {
			fmt.Println("Error sign", err)
			i--
			continue
		}
		// fmt.Println(req)
		// reqs[i] = req
		req, _ = json.Marshal(req)
		// fmt.Println(req)
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

		// sign
		// signs[i] = sign
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
		err = writer.Flush()
		if err != nil {
			log.Fatalf("Failed to flush writer: %v", err)
		}

	}
	file.Close()
	wet := time.Now().UnixNano() / 1e6
	rst := time.Now().UnixNano() / 1e6
	file, err = os.Open(filename)
	if err != nil {
		log.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// create bufio.Scanner
	scanner := bufio.NewScanner(file)

	for i := 0; i < count; i++ {
		if scanner.Scan() {
			req := []byte{}
			json.Unmarshal(scanner.Bytes(), &req)
			if scanner.Scan() {

				sign := []byte{}
				json.Unmarshal(scanner.Bytes(), &sign)
				// fmt.Println(reqs[i])
				// fmt.Println(req)
				fmt.Println(sm2.Sm2Verify(sign, pk, req))
			} else {
				log.Fatalf("Failed to read signature to file: %v %d", err, i)
				i--
				continue
			}
		} else {
			log.Fatalf("Failed to read requests to file: %v %d", err, i)
			i--
			continue
		}
	}
	ret := time.Now().UnixNano() / 1e6

	fmt.Printf("gen sign write time: %f ms\n", float64(wet-wst)/float64(count))
	fmt.Printf("read verify time   : %f ms\n", float64(ret-rst)/float64(count))
}

// TestJson: test read and write json files
func TestJson(t *testing.T) {
	count := 1000000
	length := 100
	sk, pk, _ := sm2.Sm2KeyGen(rand.Reader)
	req := common.GenerateSecureRandomByteSlice(length)
	sign, _ := sm2.Sm2Sign(sk, pk, req)
	st := time.Now().UnixNano()
	for i := 0; i < count; i++ {
		rj, err := json.Marshal(req)
		if err != nil {
			log.Fatalf("Failed to write requests to file: %v", err)
			i--
			continue
		}
		sj, err := json.Marshal(sign)
		if err != nil {
			log.Fatalf("Failed to write requests to file: %v", err)
			i--
			continue
		}

		req := []byte{}
		json.Unmarshal(rj, &req)

		sign := []byte{}
		json.Unmarshal(sj, &sign)
	}
	et := time.Now().UnixNano()
	mst := (et - st) / 1e6
	fmt.Println(mst)
	fmt.Println(float64(mst) / float64(count))
}
