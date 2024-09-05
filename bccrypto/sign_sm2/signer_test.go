package ssm2_test

import (
	"common"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"fmt"
	"os"
	ssm2 "ssm2"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"
)

// TestSigner: test the signer works
func TestSigner(t *testing.T) {

	num := 4
	count := 100
	signers := ssm2.NewSigners(4)
	for i := 0; i < num; i++ {
		fmt.Println(signers[i])
	}

	msg := common.GenerateSecureRandomByteSlice(128)

	sign := signers[1].Sign(msg)

	fmt.Println(sign)

	signs := make([][]byte, count)
	msgs := make([][]byte, count)
	for i := 0; i < count; i++ {
		msgs[i] = common.GenerateSecureRandomByteSlice(128)
		signs[i] = signers[1].Sign(msgs[i])
	}
	start := time.Now()
	for j := 0; j < count; j++ {
		// for i := 0; i < num; i++ {
		// fmt.Println(signs[j])
		if !signers[0].VerifySign(signers[1].ID, signs[j], msgs[j]) {
			fmt.Println(false)
		}
		// }
	}

	fmt.Println(time.Since(start))
	dura := float64(time.Since(start)) / float64(time.Millisecond)
	fmt.Printf("total verify count 	: %d \n", count*num)
	fmt.Printf("total verify time 	: %.3f ms\n", dura)
	fmt.Printf("per verify time  	: %.3f ms\n", dura/float64(count*num))
}

// TestStoreAndGetPK: test the function of store and get pk
func TestStoreAndGetPK(t *testing.T) {
	signers := ssm2.NewSigners(1)[0]
	signers.StorePK()
	fmt.Println(signers.Pk)
	pk := signers.GetPKFromFile()
	fmt.Println(pk)
}

// TestConsumeTime: test the consume time of sign and verify
func TestConsumeTime(t *testing.T) {

	count_out := 33
	// data := make([][8]interface{}, count_out)

	filename := "sign_data_pbft.xlsx"
	sheetName := "Sheet1"

	// check whether the file exists
	var file *excelize.File
	if _, err := os.Stat(filename); os.IsNotExist(err) {

		// if the file does not exist, create a new file
		file = excelize.NewFile()
		rowKey := []interface{}{"Reqest number", "BatchSize", "Total time", "View number", "Throughput", "Latency"}
		if err := file.SetSheetRow(sheetName, fmt.Sprintf("A%d", 1), &rowKey); err != nil {
			fmt.Println("Error setting row:", err)
			return
		}
	} else {
		// if the file exists, open the existing file
		file, err = excelize.OpenFile(filename)
		if err != nil {
			fmt.Println("Error opening file:", err)
			return
		}
	}

	for m := 26; m < count_out+1; m++ {

		// get existing data
		rows, err := file.GetRows(sheetName)
		if err != nil {
			fmt.Println("Error getting row:", err)
			return
		}

		signerNum := m*3 + 1
		signers := ssm2.NewSigners(signerNum)

		msg := common.GenerateSecureRandomByteSlice(128)
		sign := signers[1].Sign(msg)

		sign_num := 2*signerNum + 1
		verify_num := sign_num * signerNum

		st1 := time.Now()
		for m := 0; m < 1000; m++ {
			signers[1].Sign(msg)
		}
		ct1 := float64(time.Since(st1)) / float64(time.Millisecond)
		st2 := time.Now()
		for m := 0; m < 1000; m++ {
			if !signers[0].VerifySign(signers[1].ID, sign, msg) {
				fmt.Println(false)
			}
		}
		ct2 := float64(time.Since(st2)) / float64(time.Millisecond)

		st3 := time.Now()
		for l := 0; l < 10; l++ {
			for j := 0; j < (2*signerNum+1)*signerNum; j++ {
				// for i := 0; i < num; i++ {
				// fmt.Println(signs[j])
				if !signers[0].VerifySign(signers[1].ID, sign, msg) {
					fmt.Println(false)
				}
				// }
			}
			for j := 0; j < (2*signerNum+1)*signerNum; j++ {
				// for i := 0; i < num; i++ {
				// fmt.Println(signs[j])
				if !signers[0].VerifySign(signers[1].ID, sign, msg) {
					fmt.Println(false)
				}
				// }
			}
		}
		ct3 := float64(time.Since(st3)) / float64(time.Millisecond)
		// fmt.Printf("total verify count 	: %d \n", num*num)
		// fmt.Printf("total verify time 	: %.3f ms\n", dura)
		// fmt.Printf("per verify time  	: %.3f ms\n", dura/float64(num*num))

		// data to append
		data := []interface{}{
			signerNum,
			ct1 / 1000,
			ct2 / 1e3,
			sign_num,
			verify_num,
			ct3 / 10,
		} // new row data
		fmt.Println(data...)
		if err := file.SetSheetRow(sheetName, fmt.Sprintf("A%d", len(rows)+1), &data); err != nil {
			fmt.Println("Error setting row:", err)
			return
		}

		// save file
		if err := file.SaveAs(filename); err != nil {
			fmt.Println("Error saving file:", err)
			return
		}

		fmt.Println("Excel file saved successfully.", signerNum)
	}
}

// TestRSASign: test the consume time of rsa
func TestRSASign(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		fmt.Println("Error generating RSA key:", err)
		return
	}

	// 生成要签名的数据
	message := []byte("this is a secret message")
	hash := sha256.New()
	hash.Write(message)
	hashed := hash.Sum(nil)
	signature, _ := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	count := 1000
	// 签名
	st1 := time.Now()
	for i := 0; i < count; i++ {
		_, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
		if err != nil {
			fmt.Println("Error signing message:", err)
			return
		}
	}

	ct1 := float64(time.Since(st1)) / float64(time.Millisecond)

	// 验证签名
	publicKey := &privateKey.PublicKey

	st2 := time.Now()
	for i := 0; i < count; i++ {
		err = rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed, signature)
		if err != nil {
			fmt.Println("Error verifying signature:", err)
		}
	}
	ct2 := float64(time.Since(st2)) / float64(time.Millisecond)
	fmt.Printf("time per sign	: %fms\n", ct1/float64(count))
	fmt.Printf("time per verify	: %fms\n", ct2/float64(count))
}

// TestRSASign: test the consume time of ECDSA
func TestECDSASign(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		fmt.Println("Error generating RSA key:", err)
		return
	}

	// 生成要签名的数据
	message := []byte("this is a secret message")
	hash := sha256.New()
	hash.Write(message)
	hashed := hash.Sum(nil)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed)
	if err != nil {
		fmt.Println("Error signing message:", err)
		return
	}

	count := 1000
	// 签名

	st1 := time.Now()
	for i := 0; i < count; i++ {
		_, _, err := ecdsa.Sign(rand.Reader, privateKey, hashed)
		if err != nil {
			fmt.Println("Error sign message:", err)
			return
		}
	}

	ct1 := float64(time.Since(st1)) / float64(time.Millisecond)

	// 验证签名
	publicKey := &privateKey.PublicKey
	fmt.Println(privateKey)
	fmt.Println(publicKey)

	st2 := time.Now()
	for i := 0; i < count; i++ {
		if !ecdsa.Verify(publicKey, hashed, r, s) {
			fmt.Println("Error verifying signature")
		}
	}
	ct2 := float64(time.Since(st2)) / float64(time.Millisecond)
	fmt.Printf("time per sign	: %fms\n", ct1/float64(count))
	fmt.Printf("time per verify	: %fms\n", ct2/float64(count))
}
