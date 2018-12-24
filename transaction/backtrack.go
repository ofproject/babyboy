package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common/queue"
	"github.com/babyboy/core/types"
	"encoding/json"
	"fmt"
	"log"
	"sort"
)

func (tr *Transaction) BackTrackingUTXO(newUnit types.Unit) {
	//tr.mux.Lock()
	//defer tr.mux.Unlock()

	fmt.Println()

	putxo := boydb.GetDbInstance().GetPendingUTXOByAuthor(newUnit.Authors[0].Address)
	if len(putxo) == 0 {
		log.Println("回溯结果: ", "正常")
		return
	}
	log.Println("回溯起始单元: ", putxo[0].UnitHash.String())

	trackQueue := queue.New()
	trackQueue.Push(putxo[0])

	backTrackResult := false

	for !trackQueue.Empty() {
		utxo := trackQueue.Front().(types.UTXO)
		curUnit, err := boydb.GetDbInstance().GetUnitByHash(utxo.UnitHash)
		if err != nil {
			log.Println("UTXO对应的单元不存在")
			break
		}

		trackQueue.Pop()

		log.Println("当前单元的Input个数: ", len(curUnit.Messages[0].Payload.Inputs))

		chs := make([]chan ResultBack, len(curUnit.Messages[0].Payload.Inputs))
		for i := 0; i < len(curUnit.Messages[0].Payload.Inputs); i++ {
			chs[i] = make(chan ResultBack, 2)
			tr.isInputHasSpent(curUnit, i, chs[i])
		}

		isBreak := false
		for _, ch := range chs {
			result := <-ch
			if result.stable {
				if result.exist {
					log.Println("稳定单元 UTXO 存在")
					backTrackResult = true
					isBreak = true
				} else {
					log.Println("稳定单元 UTXO 不存在")
					strByte, _ := json.Marshal(result.utxo)
					log.Println(string(strByte))
					backTrackResult = false
					isBreak = true
				}
			} else {
				log.Println("不稳定单元: 继续回溯")
				trackQueue.Push(result.utxo)
			}
		}

		if isBreak {
			break
		}
	}

	if backTrackResult {
		log.Println("回溯结果: ", "正常")
	} else {
		log.Println("回溯结果: ", "异常")
	}

	if !backTrackResult {
		tr.reBuildPendingPool(newUnit)
	}

	fmt.Println()
}

func (tr *Transaction) reBuildPendingPool(newUnit types.Unit) {
	log.Println("Rebuild Pending UTXO")
	utxos := boydb.GetDbInstance().GetPendingUTXOByAuthor(newUnit.Authors[0].Address)
	pendingUnits := make(types.Units, len(utxos))
	for _, u := range utxos {
		unit, err := boydb.GetDbInstance().GetUnitByHash(u.UnitHash)
		if err != nil {
			log.Println(err)
			continue
		}
		pendingUnits = append(pendingUnits, unit)
	}
	pUnSpent := boydb.GetDbInstance().GetAllPendingUnSpent(newUnit.Authors[0].Address)
	for _, u := range pUnSpent {
		boydb.GetDbInstance().DelPendingUTXO(newUnit.Authors[0].Address, u)
		strByte, _ := json.Marshal(u)
		log.Println(string(strByte))
	}
	sort.Sort(pendingUnits)
	for _, u := range pendingUnits {
		tr.PendingTx(u)
	}
}

func (tr *Transaction) isInputHasSpent(curUnit types.Unit, index int, ch chan ResultBack) {
	utype := curUnit.Messages[0].Payload.Inputs[index].Type
	preUTXO := types.UTXO{}

	switch utype {
	case "wc":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		output := input.Output
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, 0, output, input.Type)

		log.Println("当前单元: ", curUnit.IsStable)
		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(curUnit.Authors[0].Address, preUTXO)

		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	case "mc":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		output := input.Output
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, 0, output, input.Type)

		log.Println("当前单元: ", curUnit.IsStable)
		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(curUnit.Authors[0].Address, preUTXO)

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
		output := input.Output
		preUTXO = types.NewUTXO(inputUnit.Hash, messageIdx, outputIdx, output, "")

		log.Println("当前单元: ", curUnit.IsStable)
		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(curUnit.Authors[0].Address, preUTXO)

		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	}
}
