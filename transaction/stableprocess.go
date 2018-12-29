package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/core/types"
	"encoding/json"
	"log"
	"babyboy-dag/boydb"
)

type StableProcess struct {
	db *boydb.DatabaseManager
}

func NewStableProcess() *StableProcess {
	return &StableProcess{}
}

func (sp *StableProcess) HandleUnit(newUnit types.Unit) ([]types.Commission, bool, error) {

	commissions := make([]types.Commission, 0)

	for i := 0; i < len(newUnit.Messages); i++ {
		curMessage := newUnit.Messages[i]

		for j := 0; j < len(curMessage.Payload.Inputs); j++ {
			uType := curMessage.Payload.Inputs[j].Type

			pendingSpent := types.UTXO{}

			switch uType {
			case "wc":
				input := curMessage.Payload.Inputs[j]
				inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
				if err != nil {
					log.Println("未找到输入来源的单元数据")
					return commissions, false, ErrNotFindFrom
				}
				output := input.Output
				pendingSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: 0, OutputIndex: 0, Output: output, Type: input.Type}
				break
			case "mc":
				input := curMessage.Payload.Inputs[j]
				inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
				if err != nil {
					log.Println("未找到输入来源的单元数据")
					return commissions, false, ErrNotFindFrom
				}
				output := input.Output
				pendingSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: 0, OutputIndex: 0, Output: output, Type: input.Type}
				break
			case "":
				input := curMessage.Payload.Inputs[j]
				inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
				if err != nil {
					log.Println("未找到输入来源的单元数据")
					return commissions, false, ErrNotFindFrom
				}
				messageIdx := 0
				outputIdx := curMessage.Payload.Inputs[j].OutputIndex
				output := input.Output
				pendingSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: ""}
				break
			}

			if boydb.GetDbInstance().IsExistUnspentOutput(pendingSpent) {
				boydb.GetDbInstance().DelUnspentOutput(newUnit.Authors[0].Address, pendingSpent)
			} else {
				log.Println("稳定的UTXO不存在,可能被其他交易使用")
				sp.print(pendingSpent)
				log.Println("双花交易: ", newUnit.MainChainIndex, " ", newUnit.IsOnMainChain, "", newUnit.Hash.String())
				return commissions, false, nil
			}
		}

		for z := 0; z < len(curMessage.Payload.Outputs); z++ {
			address := curMessage.Payload.Outputs[z].Address
			amount := curMessage.Payload.Outputs[z].Amount

			unSpent := types.NewUTXO(newUnit.Hash, i, z, types.Output{Amount: amount, Address: address}, "")
			boydb.GetDbInstance().DelPendingUnspentOutput(unSpent)
			commission := types.NewCommission(address, unSpent)
			commissions = append(commissions, commission)
		}
	}

	minerCommission := sp.distributionMinerCommission(newUnit)
	commissions = append(commissions, minerCommission)

	log.Println()
	log.Println("处理完稳定点扩展", newUnit.MainChainIndex, " ", newUnit.IsOnMainChain, "", newUnit.Hash.String())
	log.Println()

	return commissions, true, nil
}

func (sp *StableProcess) distributionMinerCommission(newUnit types.Unit) types.Commission {

	minHashAuthor := newUnit.SubStableAuthor

	minerUnSpent := types.NewUTXO(newUnit.Hash, 0, 0, types.NewOutput(minHashAuthor, newUnit.HeadersCommission), "mc")

	return types.NewCommission(minHashAuthor, minerUnSpent)
}

func (sp *StableProcess) print(u types.UTXO) {
	strByte, _ := json.Marshal(u)
	log.Println(string(strByte))
}
