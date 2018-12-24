package transaction

import (
	"github.com/babyboy/core/types"
	"encoding/json"
	"log"
	"math/big"
)

func (tr *Transaction) ValidUnitInputsAndOutputs(unit types.Unit) error {
	if len(unit.Messages) <= 0 {
		return ErrUnitInfo
	}
	for i := 0; i < len(unit.Messages); i++ {
		if len(unit.Messages[i].Payload.Inputs) <= 0 {
			return ErrUnitInfo
		}
		for j := 0; j < len(unit.Messages[i].Payload.Inputs); j++ {
			if unit.Messages[i].Payload.Inputs[i].Output.Amount <= 0 {
				return ErrUnitInfo
			}
		}
		if len(unit.Messages[i].Payload.Outputs) <= 0 {
			return ErrUnitInfo
		}
		for j := 0; j < len(unit.Messages[i].Payload.Outputs); j++ {
			if unit.Messages[i].Payload.Outputs[j].Amount <= 0 {
				return ErrUnitInfo
			}
		}
	}
	if len(unit.Authors) <= 0 {
		return ErrUnitInfo
	}
	return nil
}

func (tr *Transaction) ValidUTXOAmount(unit types.Unit) error {

	inputAmount := 0
	for i := 0; i < len(unit.Messages[0].Payload.Inputs); i++ {
		// 根据索引找到前一个单元中对应的UTXO
		inputAmount = inputAmount + unit.Messages[0].Payload.Inputs[i].Output.Amount
	}

	outputAmount := 0
	for i := 0; i < len(unit.Messages[0].Payload.Outputs); i++ {
		outputAmount = outputAmount + unit.Messages[0].Payload.Outputs[i].Amount
	}
	outputAmount = outputAmount + unit.PayloadCommission + unit.HeadersCommission

	if inputAmount != outputAmount {
		return ErrUnitAmountNoEqual
	}
	return nil
}

func (tr *Transaction) ValidAuthorAddress(unit types.Unit) error {
	targetAddress := unit.Authors[0]
	for i := 0; i < len(unit.Messages[0].Payload.Inputs); i++ {
		address := unit.Messages[0].Payload.Inputs[i].Output.Address
		if address.String() != targetAddress.Address.String() {
			return ErrUnitAuthorAddress
		}
	}
	return nil
}

// 获取矿工佣金
func (tr *Transaction) GetMinerCommission(u types.Unit) *big.Int {
	jsonByte, err := json.Marshal(u)
	if err != nil {
		log.Println(err)
	}
	commission := new(big.Int).Sub(big.NewInt(int64(len(jsonByte))), tr.GetWitnessCommission(u))

	return commission
}

// 获取见证人佣金
func (tr *Transaction) GetWitnessCommission(u types.Unit) *big.Int {
	jsonByte, err := json.Marshal(u.Messages)
	if err != nil {
		log.Println(err)
	}
	commission := new(big.Int).SetInt64(int64(len(jsonByte)))

	return commission
}
