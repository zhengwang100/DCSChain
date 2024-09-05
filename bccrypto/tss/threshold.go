package tss

import (
	"encoding/json"

	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/pairing/bn256"
	"go.dedis.ch/kyber/v3/share"
	"go.dedis.ch/kyber/v3/sign/bdn"
	"go.dedis.ch/kyber/v3/sign/tbls"
)

// Signer: the threshold signer
type Signer struct {
	Suite      *bn256.Suite    // the instance which encapsulates the BN256 curve
	PrivateKey *share.PriShare // the unshared private key, all nodes' are different
	PublicKey  *share.PubPoly  // the shared public key, all nodes' are the same
	SignNum    int             // the number of all nodes
	Threshold  int             // the min number of nodes to sign a same message
}

// NewSigners: get the signers of (sigerNum, threshold) threshold sign
//
//	suite.G2() is g
//	suite.G2().Point().Base() b
//	commits := make([]kyber.Point, p.Threshold())
//	for i := range commits {
//		commits[i] = p.g.Point().Mul(p.coeffs[i], b)
//	}
//
// params:
// -signerNum: the number of signers that need to be generated
// -threshold: set the threshold to at least the number of signers required to recover the overall signature
// return slice of generated new signers
func NewSigners(signerNum int, threshold int) []*Signer {
	signers := make([]*Signer, signerNum)
	suite := bn256.NewSuite()
	secret := suite.G1().Scalar().Pick(suite.RandomStream())
	priPoly := share.NewPriPoly(suite.G2(), threshold, secret, suite.RandomStream())
	pubPoly := priPoly.Commit(suite.G2().Point().Base())
	for i, x := range priPoly.Shares(signerNum) {
		signers[i] = &Signer{
			Suite:      suite,
			PrivateKey: x,       // assign the private key to node i
			PublicKey:  pubPoly, // assign the public key to node i
			SignNum:    signerNum,
			Threshold:  threshold,
		}
	}
	return signers
}

// ThresholdSign: threshold sign the message in byte slice
// params:
// - msg: the message that need to be signed
// return the signature and error
func (s *Signer) ThresholdSign(msg []byte) ([]byte, error) {
	return tbls.Sign(s.Suite, s.PrivateKey, msg)
}

// CombineSig: combine the threshold partial signature to a complete signature
// params:
// - msg: 		the signed message
// - sigShares: the collected shared signature
// return the recovered sginature and error
func (s *Signer) CombineSig(msg []byte, sigShares [][]byte) ([]byte, error) {
	return tbls.Recover(s.Suite, s.PublicKey, msg, sigShares, s.Threshold, s.SignNum)
}

// ThresholdSignVerify: use the shared public key to verify the digital signature
// params:
// - msg: the signed message
// - sig: the recovered signature which need to be verify
// return whether the signature is valid
func (s *Signer) ThresholdSignVerify(msg []byte, sig []byte) bool {
	err := bdn.Verify(s.Suite, s.PublicKey.Commit(), msg, sig)
	if err == nil {
		return true
	} else {
		return false
	}
}

// Encode: encode the signer self private key to []byte
func (s *Signer) Encode() []byte {
	pri, err := s.PrivateKey.V.MarshalBinary()
	if err != nil {
		return nil
	}
	js, err := json.Marshal(SignerJson{s.PrivateKey.I, pri, s.SignNum, s.Threshold})
	if err != nil {
		return nil
	}
	return js
}

// Encode: decode []byte to the signer self private key
func (s *Signer) Decode(data []byte) error {
	var signerJson SignerJson
	err := json.Unmarshal(data, &signerJson)
	if err != nil {
		return err
	}
	s.SignNum = signerJson.SignNum
	s.Threshold = signerJson.Threshold
	s.PrivateKey.I = signerJson.I
	kyber.Scalar.SetBytes(s.PrivateKey.V, signerJson.V)
	return nil
}
