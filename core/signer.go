package core

import (
	"babyboy/common"
	"babyboy/core/types"
	"babyboy/crypto"
	"encoding/json"
	"fmt"
	"log"
)

type Signer struct {
}

func NewSigner() *Signer {
	signer := &Signer{}

	return signer
}

func (s *Signer) VerifyUnit(unit types.Unit) bool {
	var newUnit = types.Unit{}
	newUnit = unit
	originalAddr := newUnit.Authors[0].Address
	copyData := make([]byte, 65)
	if len(unit.Authors[0].Signature) != 65 {
		log.Print("signature must be 65 bytes long")
		return false
	}

	copy(copyData, unit.Authors[0].Signature)
	if copyData[64] != 27 && copyData[64] != 28 {
		log.Println("invalid Ethereum signature (V is not 27 or 28)")
		return false
	}
	copyData[64] -= 27
	newUnit.Authors = types.Authors{}
	newUnit.Hash = common.Hash{}
	jsonStr, err := json.Marshal(newUnit)
	if err != nil {
		log.Println(err)
		return false
	}
	pubKey, RecoverErr := crypto.SigToPub(s.signHash(jsonStr), copyData)
	if RecoverErr != nil {
		fmt.Println("Recover Public key error!")
		return false
	}
	if crypto.PubkeyToAddress(*pubKey) == originalAddr {

		return true
	}
	log.Println("验证数据结果： ", false)

	return false
}

func (s *Signer) signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	return crypto.Keccak256([]byte(msg))
}
