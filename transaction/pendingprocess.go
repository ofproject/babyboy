package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/core/types"
	"encoding/json"
	"log"
	"babyboy-dag/boydb"
)

type PendingPool struct {
	PendingTx map[string]types.UTXO
}

func NewPendingPool() *PendingPool {
	return &PendingPool{}
}

func (pool *PendingPool) HandleUnit(unit types.Unit) error {

	var utxos []UtxoHelper
	db := leveldb.GetDbInstance()
	for i := 0; i < len(unit.Messages); i++ {
		curMessage := unit.Messages[i]

		for j := 0; j < len(curMessage.Payload.Inputs); j++ {
			inputUnit, err := db.GetUnitByHash(curMessage.Payload.Inputs[j].UnitHash)
			if err != nil {
				log.Println("未找到该笔交易的输入来源,请同步数据: ", unit.Hash.String())
				return ErrNotFindFrom
			}

			typeOf := curMessage.Payload.Inputs[j].Type
			messageIdx := curMessage.Payload.Inputs[j].MessageIndex
			outputIdx := curMessage.Payload.Inputs[j].OutputIndex
			output := curMessage.Payload.Inputs[j].Output
			futureSpent := types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: typeOf}

			if inputUnit.IsStable {

				isExist := boydb.GetDbInstance().IsExistUnspentOutput(futureSpent)
				if !isExist {
					log.Println("该单元的未花费在Pending池中未找到")
					pool.print(futureSpent)
					return nil
				}

				utxos = append(utxos, UtxoHelper{UTXO: futureSpent, IsStable: inputUnit.IsStable})
			} else {
				isExist := db.IsExistPendingUTXO(futureSpent)
				if !isExist {
					log.Println("该单元的未花费在Pending池中未找到")
					pool.print(futureSpent)
					return nil
				}

				utxos = append(utxos, UtxoHelper{UTXO: futureSpent, IsStable: inputUnit.IsStable})
			}
		}

		for _, spent := range utxos {
			if !spent.IsStable {
				db.DelPendingUnspentOutput(spent.UTXO)
			}
		}


		for z := 0; z < len(curMessage.Payload.Outputs); z++ {
			address := curMessage.Payload.Outputs[z].Address
			amount := curMessage.Payload.Outputs[z].Amount

			if address == unit.Authors[0].Address {
				unSpent := types.UTXO{UnitHash: unit.Hash, MessageIndex: i,
					OutputIndex: z, Output: types.Output{Amount: amount, Address: address}, Type: ""}

				db.SavePendingUTXO(address, unSpent)
			}
		}
	}

	return nil
}

func (pool *PendingPool) print(u types.UTXO) {
	strByte, _ := json.Marshal(u)
	log.Println(string(strByte))
}
