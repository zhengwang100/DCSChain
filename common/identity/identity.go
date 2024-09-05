package identity

// PubID: include name and public key which are known to all nodes in system
type PubID struct {
	Name   string // the name publicID for everyone to know
	PubKey []byte // the pubkey
}

// PrivID: inlucde PubID and private key which is only known to self
type PrivID struct {
	ID         PubID       // the publik id
	Address    chan []byte // the node self address in network
	PrivateKey []byte      // the private key
}
