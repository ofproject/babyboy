package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/queue"
	"github.com/babyboy/core/types"
	"github.com/babyboy/dag"
	"github.com/babyboy/dag/memdb"
	"encoding/json"
	"errors"
	"log"
	"math/big"
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
		//commission := inputUnit.PayloadCommission
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, 0, input.Output, input.Type)

		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(curUnit.Authors[0].Address, preUTXO)
		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	case "mc":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		//commission := inputUnit.HeadersCommission
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, 0, input.Output, input.Type)

		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(curUnit.Authors[0].Address, preUTXO)
		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	case "":
		input := curUnit.Messages[0].Payload.Inputs[index]
		inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
		if err != nil {
			log.Println(err)
		}
		messageIdx := big.NewInt(0)
		outputIdx := curUnit.Messages[messageIdx.Int64()].Payload.Inputs[index].OutputIndex
		//amount := inputUnit.Messages[messageIdx.Int64()].Payload.Outputs[outputIdx.Int64()].Amount
		preUTXO = types.NewUTXO(inputUnit.Hash, 0, outputIdx, input.Output, input.Type)

		existStableUTXO := boydb.GetDbInstance().IsExistUnspentOutput(curUnit.Authors[0].Address, preUTXO)
		ch <- ResultBack{stable: inputUnit.IsStable, exist: existStableUTXO, utxo: preUTXO}
		break
	}
}

func (tr *Transaction) GetMainChainMerkleRoot() []byte {
	db := boydb.GetDbInstance()
	pdb := memdb.GetParentMemDBInstance()
	wdb := memdb.GetWitnessMemDBInstance()

	graphInfo := dag.NewGraphInfoGetter(db, pdb.GetParentsAsHash(), wdb.GetWitnessesAsHash())
	startMci := graphInfo.GetLastStableBallMCI()
	sUnits, _ := graphInfo.GetMissingUnits(startMci, 0)

	var list []common.Content
	for _, value := range sUnits {
		list = append(list, value)
	}

	t, err := common.NewTree(list)
	if err != nil {
		log.Fatal(err)
	}

	// Get the Merkle Root of the tree
	mr := t.MerkleRoot()

	return mr
}

func (tr *Transaction) IsUnitInMerkleTree(unit types.Unit) (bool, error) {
	db := boydb.GetDbInstance()
	pdb := memdb.GetParentMemDBInstance()
	wdb := memdb.GetWitnessMemDBInstance()

	graphInfo := dag.NewGraphInfoGetter(db, pdb.GetParentsAsHash(), wdb.GetWitnessesAsHash())
	startMci := graphInfo.GetLastStableBallMCI()
	sUnits, _ := graphInfo.GetMissingUnits(startMci, 0)

	var list []common.Content
	for _, value := range sUnits {
		list = append(list, value)
	}

	t, err := common.NewTree(list)
	if err != nil {
		log.Fatal(err)
	}

	vc, err := t.VerifyContent(unit)
	if err != nil {
		log.Fatal(err)
	}

	return vc, err
}

func (tr *Transaction) VerifyMessageInputs(unit types.Unit) error {

	var utxos []UtxoHelper

	curMessage := unit.Messages[0]
	//strByte, _ := json.Marshal(curMessage)
	//log.Println(string(strByte))
	for j := 0; j < len(curMessage.Payload.Inputs); j++ {
		inputUnit, err := tr.db.GetUnitByHash(curMessage.Payload.Inputs[j].UnitHash)
		if err != nil {
			log.Println("未找到该笔交易的输入来源,请同步数据: ", unit.Hash.String())
			return errors.New("未找到该笔交易的输入来源,请同步数据")
		}

		messageIdx := curMessage.Payload.Inputs[j].MessageIndex
		outputIdx := curMessage.Payload.Inputs[j].OutputIndex
		output := curMessage.Payload.Inputs[j].Output
		futureSpent := types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: ""}

		if inputUnit.IsStable {
			switch curMessage.Payload.Inputs[j].Type {
			case "wc":
				input := curMessage.Payload.Inputs[j]
				inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
				if err != nil {
					log.Println("未找到输入来源的单元数据")
					return err
				}
				output := curMessage.Payload.Inputs[j].Output
				messageIdx := curMessage.Payload.Inputs[j].MessageIndex
				outputIdx := curMessage.Payload.Inputs[j].OutputIndex
				futureSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: curMessage.Payload.Inputs[j].Type}
				break
			case "mc":
				input := curMessage.Payload.Inputs[j]
				inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
				if err != nil {
					log.Println("未找到输入来源的单元数据")
					return err
				}
				output := curMessage.Payload.Inputs[j].Output
				messageIdx := curMessage.Payload.Inputs[j].MessageIndex
				outputIdx := curMessage.Payload.Inputs[j].OutputIndex
				futureSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: curMessage.Payload.Inputs[j].Type}
				break
			case "":
				input := curMessage.Payload.Inputs[j]
				inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
				if err != nil {
					log.Println("未找到输入来源的单元数据")
					return err
				}
				messageIdx := curMessage.Payload.Inputs[j].MessageIndex
				outputIdx := curMessage.Payload.Inputs[j].OutputIndex
				output := input.Output
				futureSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: ""}
				break
			}

			isExist := boydb.GetDbInstance().IsExistUnspentOutput(unit.Authors[0].Address, futureSpent)
			if !isExist {
				strByte, _ := json.Marshal(futureSpent)
				log.Println(string(strByte))
				return errors.New("该单元的未花费输出不存在,请重新同步数据")
			}

			utxos = append(utxos, UtxoHelper{Address: unit.Authors[0].Address, UTXO: futureSpent, IsStable: inputUnit.IsStable})
		} else {
			input := curMessage.Payload.Inputs[j]
			inputUnit, err := boydb.GetDbInstance().GetUnitByHash(input.UnitHash)
			if err != nil {
				log.Println("未找到输入来源的单元数据")
				return err
			}
			messageIdx := curMessage.Payload.Inputs[j].MessageIndex
			outputIdx := curMessage.Payload.Inputs[j].OutputIndex
			output := input.Output
			futureSpent = types.UTXO{UnitHash: inputUnit.Hash, MessageIndex: messageIdx, OutputIndex: outputIdx, Output: output, Type: ""}
			isExist := tr.db.IsExistPendingUTXO(unit.Authors[0].Address, futureSpent)
			if !isExist {
				log.Println(futureSpent)
				log.Println("该单元的未花费在Pending池中未找到")
				return nil
			}

			utxos = append(utxos, UtxoHelper{Address: unit.Authors[0].Address, UTXO: futureSpent, IsStable: inputUnit.IsStable})
		}
	}

	return nil
}
