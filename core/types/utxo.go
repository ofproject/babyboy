package types

import (
	"github.com/babyboy/common"
	"github.com/babyboy/crypto/sha3"
	"encoding/json"
)

type UTXO struct {
	UnitHash     common.Hash
	MessageIndex int
	OutputIndex  int
	Output       Output
	Type         string
}

func (u UTXO) ToHash() common.Hash {
	jsonByte, _ := json.Marshal(u)
	hash := sha3.Sum256(jsonByte)

	return hash
}

func NewUTXO(unitHash common.Hash, messageIdx int, outputIdx int, output Output, typeOf string) UTXO {
	return UTXO{UnitHash: unitHash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: typeOf}
}

func NewEmptyUTXO() UTXO {
	return UTXO{}
}
