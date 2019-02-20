package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common/queue"
	"github.com/babyboy/core/types"
	"log"
	"babyboy-dag/boydb"
)

func (tr *Transaction) BackTrackingHasSpentUTXO(unspent types.UTXO) []types.UTXO {
	tr.mux.Lock()
	defer tr.mux.Unlock()

	stableUTXOS := make([]types.UTXO, 0)

	trackQueue := queue.New()
	trackQueue.Push(unspent)

	for !trackQueue.Empty() {
		utxo := trackQueue.Front().(types.UTXO)
		curUnit, err := boydb.GetDbInstance().GetUnitByHash(utxo.UnitHash)
		if err != nil {
			log.Println(err)
			log.Println("UTXO对应的单元不存在", utxo.UnitHash.String())
			break
		} else {
			log.Println("单元存在", curUnit.Hash.String())
		}

		trackQueue.Pop()

		// 找到UTXO对应的单元的前一个单元
		log.Println("Current Input Count: ", len(curUnit.Messages[0].Payload.Inputs))

		chs := make([]chan ResultBack, len(curUnit.Messages[0].Payload.Inputs))
		for i := 0; i < len(curUnit.Messages[0].Payload.Inputs); i++ {
			chs[i] = make(chan ResultBack, 2)
			tr.isAddInputHasSpent(curUnit, i, chs[i])
		}

		for _, ch := range chs {
			result := <-ch
			if result.stable {
				if result.exist {
					log.Println("稳定单元 UTXO 存在")
					stableUTXOS = append(stableUTXOS, result.utxo)
				} else {
					log.Println("稳定单元 UTXO 不存在")
				}
			} else {
				// log.Println("不稳定单元: 继续回溯")
				trackQueue.Push(result.utxo)
			}
		}
	}

	return stableUTXOS
}

func (tr *Transaction) isAddInputHasSpent(curUnit types.Unit, index int, ch chan ResultBack) {
	
	utype := curUnit.Messages[0].Payload.Inputs[index].Type
	preUTXO := types.UTXO{}

	switch utype {
	case "wc":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, 0, input.Output, input.Type)

		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(preUTXO)
		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	case "mc":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, 0, input.Output, input.Type)

		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(preUTXO)
		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	case "":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		messageIdx := 0
		outputIdx := curUnit.Messages[messageIdx].Payload.Inputs[index].OutputIndex
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, outputIdx, input.Output, input.Type)

		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(preUTXO)
		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	}
}
