package tss_test

import (
	"common"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
	"tss"

	"github.com/xuri/excelize/v2"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/share"
)

// TestNewSigners: test to create new signer
func TestNewSigners(t *testing.T) {
	newSigners := tss.NewSigners(4, 3)
	fmt.Println(newSigners)
	for _, s := range newSigners {
		// fmt.Println(s.Suite, s.PrivateKey, s.PublicKey)
		fmt.Println("PrivateKey", s.PrivateKey)
		fmt.Println("PublicKey", s.PublicKey)
	}
}

// TestSigners: the sign and verify with multi-signers
func TestSigners(t *testing.T) {
	msg := []byte("hello tss")
	newSigners := tss.NewSigners(4, 3)
	sig := make([][]byte, 4)
	fmt.Println(newSigners)
	for i, s := range newSigners {
		ss, err := s.ThresholdSign(msg)
		if err == nil {
			sig[i] = ss
		}
	}
	for i := 0; i < 4; i++ {
		fmt.Println(sig[i])
	}
	comSig, _ := newSigners[0].CombineSig(msg, sig[1:])
	fmt.Println(comSig)
	for i := 0; i < 4; i++ {
		fmt.Println(newSigners[0].ThresholdSignVerify(msg, comSig))
	}
	msg2 := []byte("hello test")
	newSigners2 := tss.NewSigners(4, 3)
	sig2 := make([][]byte, 4)
	for i, s := range newSigners2 {
		ss, err := s.ThresholdSign(msg2)
		if err == nil {
			sig2[i] = ss
		}
	}
	for i := 0; i < 4; i++ {
		fmt.Println(sig2[i])
	}

	comSig2, _ := newSigners2[0].CombineSig(msg2, sig2[1:])
	fmt.Println("comsig2", comSig2)
	// var comSig3 []byte
	sig3 := make([][]byte, len(sig))
	copy(sig3, sig)
	fmt.Println(len(sig3), len(sig2))
	copy(sig3[1], sig2[1]) // 修改4个签名中的一个签名
	comSig3, _ := newSigners[0].CombineSig(msg, sig3)
	fmt.Println("comsig3", comSig3)
	fmt.Println(newSigners[0].ThresholdSignVerify(msg, comSig3))
	copy(sig3[0], sig2[0])
	comSig4, _ := newSigners[0].CombineSig(msg, sig3[1:])
	fmt.Println("comsig4", comSig4)
}

// TestChanTrans: test channel transmission tss-signer
func TestChanTrans(t *testing.T) {
	msg := []byte("hello tss")
	newSigners := tss.NewSigners(4, 3)
	sig := make([][]byte, 4)
	fmt.Println(newSigners)
	for i, s := range newSigners {
		ss, err := s.ThresholdSign(msg)
		if err == nil {
			sig[i] = ss
		}
	}
	for i := 0; i < 4; i++ {
		fmt.Println(sig[i])
	}
	comSig, _ := newSigners[0].CombineSig(msg, sig[1:])
	fmt.Println(comSig)
	for i := 0; i < 4; i++ {
		fmt.Println(newSigners[0].ThresholdSignVerify(msg, comSig))
	}

	ch := make(chan []byte, 4)

	signerJson, err := json.Marshal((*newSigners[0]))
	if err != nil {
		panic(err)
	}
	fmt.Println(signerJson)
	ch <- signerJson
	tsigner := &tss.Signer{}
	tsJson := <-ch
	er := json.Unmarshal(tsJson, tsigner)
	fmt.Println(tsJson)
	if er != nil {
		panic(er)
	}
	fmt.Println(tsigner)
	fmt.Println(tsigner.ThresholdSignVerify(msg, comSig))
}

// estParamsEqual: test whether the parameters in tss are equal
func TestParamsEqual(t *testing.T) {
	newSigners1 := tss.NewSigners(4, 3)
	newSigners2 := tss.NewSigners(4, 3)
	newSigners3 := tss.NewSigners(5, 3)
	newSigners4 := tss.NewSigners(7, 4)
	for _, v := range newSigners1 {
		// fmt.Println(v.Suite.G1().Point(), v.Suite.G2().Point())
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	for _, v := range newSigners2 {
		// fmt.Println(v.Suite.G1().Point(), v.Suite.G2().Point())
		fmt.Println(v.PrivateKey.I, v.PrivateKey.V)
	}
	for _, v := range newSigners3 {
		// fmt.Println(v.Suite.G1().Point(), v.Suite.G2().Point())
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	for _, v := range newSigners4 {
		// fmt.Println(v.Suite.G1().Point(), v.Suite.G2().Point())
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	js, er := newSigners2[1].PrivateKey.V.MarshalBinary()
	fmt.Println(js, er)
	newPri := kyber.Scalar.SetBytes(newSigners1[0].PrivateKey.V, js)
	fmt.Println(newPri)
	fmt.Println(newSigners1[0].PrivateKey.V)
	// tt := share.PriShare{I: 0, V: kyber.Scalar{}}
	// var aaa []byte
	// aaa = newSigners1[1].PrivateKey.V
	// fmt.Println(aaa)
	// aa, _ := json.Marshal(newSigners1[1].PrivateKey.V)

	// ch := make(chan []byte, 4)
	// ch <- aa
	// aaa := <-ch
	// np := share.PriShare{}
	// newPri := kyber.Scalar.SetBytes(np.V, aaa)
	// fmt.Println(newSigners1[1].PrivateKey.V, newPri)
	// fmt.Println(aa, aaa)
	// json.Unmarshal(aaa, &newPri)
	// fmt.Println(newPri)
}

// TestPub: test private key of tss
func TestPri(t *testing.T) {
	newSigners1 := tss.NewSigners(4, 3)
	newSigners2 := make([]*tss.Signer, 4)
	newSigners3 := tss.NewSigners(4, 3)
	copy(newSigners2, newSigners1)
	copy(newSigners2[0:1], newSigners3[0:1])
	for _, v := range newSigners1 {
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	fmt.Println()
	for _, v := range newSigners2 {
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	fmt.Println()
	for _, v := range newSigners3 {
		fmt.Println(v.PrivateKey, v.PublicKey)
	}

	ch := make(chan []byte, 4)

	str := newSigners1[0].Encode()

	ch <- str

	err := newSigners2[0].Decode(<-ch)
	fmt.Println(err)

	fmt.Println()

	for _, v := range newSigners1 {
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	fmt.Println()
	for _, v := range newSigners2 {
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
	fmt.Println()
	for _, v := range newSigners3 {
		fmt.Println(v.PrivateKey, v.PublicKey)
	}
}

// TestPub: test public key of tss
func TestPub(t *testing.T) {
	newSigners1 := tss.NewSigners(4, 3)
	newSigners2 := make([]*tss.Signer, 4)
	newSigners3 := tss.NewSigners(4, 3)
	copy(newSigners2, newSigners1)
	copy(newSigners2[0:1], newSigners3[0:1])
	for _, v := range newSigners1 {
		fmt.Println(v.PublicKey)
	}
	fmt.Println()
	for _, v := range newSigners2 {
		fmt.Println(v.PublicKey)
	}
	fmt.Println()
	for _, v := range newSigners3 {
		fmt.Println(v.PublicKey)
	}

	// ch := make(chan []byte, 4)
	// // vJs, _ := proto.Marshal(newSigners1[0].PublicKey)
	// vJs, _ := json.Marshal(newSigners1[0].PublicKey)
	// ch <- vJs
	// vJ := <-ch

	// var v share.PubPoly
	// proto.Unmarshal(vJ, &v)
	// fmt.Println(v)
}

// TestPub: test information of public key of tss
func TestInfo(t *testing.T) {
	newSigners1 := tss.NewSigners(4, 3)
	newSigners2 := tss.NewSigners(4, 3)
	newSigners3 := tss.NewSigners(5, 3)
	newSigners4 := tss.NewSigners(7, 4)
	for _, v := range newSigners1 {
		b, com := v.PublicKey.Info()
		fmt.Println(b)
		fmt.Println(com)
	}
	fmt.Println()
	for _, v := range newSigners2 {
		b, com := v.PublicKey.Info()
		fmt.Println(b)
		fmt.Println(com)
	}
	fmt.Println()
	for _, v := range newSigners3 {
		b, com := v.PublicKey.Info()
		fmt.Println(b)
		fmt.Println(com)
	}
	fmt.Println()
	for _, v := range newSigners4 {
		b, com := v.PublicKey.Info()
		fmt.Println(b)
		fmt.Println(com)
	}
}

// TestInner: test the inner params of tss
func TestInner(t *testing.T) {
	signerNum := 4
	threshold := 3
	signers := make([]*tss.Signer, signerNum)
	suite := bn256.NewSuite()
	secret := suite.G1().Scalar().Pick(suite.RandomStream())
	rand2 := suite.RandomStream()
	priPoly := share.NewPriPoly(suite.G2(), threshold, secret, rand2)
	pubPoly := priPoly.Commit(suite.G2().Point().Base())
	for i, x := range priPoly.Shares(signerNum) {
		signers[i] = &tss.Signer{
			Suite:      suite,
			PrivateKey: x,       // assign the private key to node i
			PublicKey:  pubPoly, // assign the public key to node i
			SignNum:    signerNum,
			Threshold:  threshold,
		}
	}
	fmt.Println(secret)
	fmt.Println(suite.G2())

	fmt.Println(json.Marshal(suite.G2().Point().Base()))
	fmt.Println(suite.G2().Point().Base())

	fmt.Println()

	suite2 := bn256.NewSuite()
	secret2 := suite2.G1().Scalar().Pick(suite2.RandomStream())
	rand3 := suite2.RandomStream()
	priPoly2 := share.NewPriPoly(suite.G2(), threshold, secret2, rand3)
	pubPoly2 := priPoly2.Commit(suite.G2().Point().Base())
	for i, x := range priPoly2.Shares(signerNum) {
		signers[i] = &tss.Signer{
			Suite:      suite,
			PrivateKey: x,        // assign the private key to node i
			PublicKey:  pubPoly2, // assign the public key to node i
			SignNum:    signerNum,
			Threshold:  threshold,
		}
	}

	fmt.Println(secret2)
	fmt.Println(suite2.G2())

	fmt.Println(json.Marshal(suite2.G2().Point().Base()))
	fmt.Println(suite2.G2().Point().Base())

}

// TestInner: test time consuming to test Tss
func TestConsumeTime(t *testing.T) {
	count_out := 10
	// data := make([][8]interface{}, count_out)

	filename := "sign_data.xlsx"
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

	for m := 1; m < count_out+1; m++ {

		// get existing data
		rows, err := file.GetRows(sheetName)
		if err != nil {
			fmt.Println("Error getting row:", err)
			return
		}

		signerNum := m*3 + 1
		threshold := (signerNum-1)/3*2 + 1
		newSigners := tss.NewSigners(signerNum, threshold)

		msgLenght := 128
		message := common.GenerateSecureRandomByteSlice(msgLenght)

		count := 100
		signs := make([][]byte, threshold)
		st1 := time.Now()
		for i := 0; i < count; i++ {
			for j := 0; j < threshold; j++ {
				sign, _ := newSigners[j].ThresholdSign(message)
				signs[j] = sign
			}
		}
		ct1 := time.Since(st1)

		signCom, _ := newSigners[0].CombineSig(message, signs)
		st2 := time.Now()
		for i := 0; i < count; i++ {
			newSigners[0].CombineSig(message, signs)
		}
		ct2 := time.Since(st2)

		st3 := time.Now()
		for i := 0; i < count; i++ {
			if !newSigners[3].ThresholdSignVerify(message, signCom) {
				fmt.Println("error")
			}
		}
		ct3 := time.Since(st3)

		verify_num := signerNum * 4
		// verify_num := 4
		sign_num := signerNum * 3
		// sign_num := 3
		comb_num := 3

		st4 := time.Now()
		for i := 0; i < count/100; i++ {
			for j := 0; j < sign_num; j++ {
				newSigners[0].ThresholdSign(message)
			}
			for l := 0; l < comb_num; l++ {
				newSigners[0].CombineSig(message, signs)
			}
			for k := 0; k < verify_num; k++ {
				if !newSigners[3].ThresholdSignVerify(message, signCom) {
					fmt.Println("error")
				}
			}
		}

		ct4 := time.Since(st4)

		// data to append
		data := []interface{}{
			signerNum,
			float64(ct1/(time.Duration(count)*time.Duration(threshold))) / 1e6,
			float64(ct2/time.Duration(count)) / 1e6,
			float64(ct3/time.Duration(count)) / 1e6,
			sign_num,
			comb_num,
			verify_num,
			float64(ct4/time.Duration(count/100)) / 1e6,
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

	// fmt.Println("sign time		: ", ct1/(time.Duration(count)*time.Duration(threshold)))
	// fmt.Println("combine time	: ", ct2/time.Duration(count))
	// fmt.Println("verify time	: ", ct3/time.Duration(count))
	// fmt.Println("one round		: ", ct4/time.Duration(count/100))
}
