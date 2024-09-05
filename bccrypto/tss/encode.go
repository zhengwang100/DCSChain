package tss

type SignerJson struct {
	I         int
	V         []byte
	SignNum   int
	Threshold int

	// G       kyber.Group
	// B       kyber.Point
	// Commits []kyber.Point
}
