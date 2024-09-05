package common

// PubID: include name and public key which are known to all nodes in system
type PubID struct {
	Name   string // the unique identification of node instance, which is the same as the serverID
	PubKey []byte // the public key of node
}

// PrivID: inlucde PubID and private key which is only known to self
type PrivID struct {
	ID         PubID       // the pubID corresponding to the private id
	Address    chan []byte // the node address
	PrivateKey []byte      // the private key
}

// type PKJson struct {
// 	CurveParams []byte
// 	Pubkey      []byte
// }

// func (pk PubID) Encode() []byte {
// 	pubKeyJson, err := json.Marshal(pk.PubKey)
// 	if err != nil {
// 		return nil
// 	}
// 	curveSJson, err := json.Marshal(pk.PubKey.Curve.Params())
// 	if err != nil {
// 		return nil
// 	}
// 	newPKJson := PKJson{
// 		CurveParams: curveSJson,
// 		Pubkey:      pubKeyJson,
// 	}
// 	res, err := json.Marshal(newPKJson)
// 	if err != nil {
// 		return nil
// 	}
// 	return res
// }

// func (pk PubID) Decode(pkJson []byte) error {
// 	var newPKJson PKJson
// 	err := json.Unmarshal(pkJson, &newPKJson)
// 	if err != nil {
// 		return err
// 	}

// 	fmt.Println(newPKJson.CurveParams)
// 	var newCurveParams sm2curve.CurveParams
// 	err = json.Unmarshal(newPKJson.CurveParams, &newCurveParams)
// 	if err != nil {
// 		return err
// 	}

// 	pukJson := newPKJson.Pubkey
// 	fmt.Println(pukJson)
// 	var puk sm2.PublicKey
// 	err = json.Unmarshal(pukJson, &puk)
// 	if err != nil {
// 		// fmt.Println(1111111, newPKJson.Pubkey)
// 		return err
// 	}
// 	pk.PubKey.Curve = &newCurveParams
// 	fmt.Println("123333", pk.PubKey)
// 	return nil
// }
