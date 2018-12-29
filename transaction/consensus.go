package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/core/types"
	"github.com/babyboy/dag"
	"github.com/babyboy/dag/memdb"
	"log"
	"sync"
	"babyboy-dag/boydb"
)

type MainChain struct {
	mutex sync.Mutex
}

func NewMainChain() *MainChain {
	mc := &MainChain{}
	mc.mutex = sync.Mutex{}

	return mc
}

func (mc *MainChain) UpdateMainChain(tran *Transaction, unit types.Unit) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()

	unit.ResetStableState()
	tran.db.SaveUnitToDb(unit)
	pdb := memdb.GetParentMemDBInstance()
	pdb.SaveNewTip(unit.Hash)

	log.Println("接收共识准备处理的单元: ", unit.Hash.String())
	mainChain := dag.NewMainChainUpdater(tran, &unit, pdb.GetDagAllTips())
	//start := time.Now()
	for {
		canExtend, units := mainChain.StableBallCanExtend()
		if !canExtend {
			break
		}
		//start := time.Now()
		mc.handleStableUnits(tran, units, mainChain)
	}

	return nil
}

func (mc *MainChain) printChainInfo(unit types.Unit) {
	log.Println("UnitHash: ", unit.Hash.String())
	log.Println("MCI: ", unit.MainChainIndex)
	log.Println("IsOnMainChain: ", unit.IsOnMainChain)
}

func (mc *MainChain) handleStableUnits(tran *Transaction, units types.Units, mainChain *dag.MainChainUpdater) {

	allCommissions := make([]types.Commission, 0)

	Len := len(units)

	validUnits := types.Units{}
	for i := 0; i < Len; i++ {
		tUnit := units[i]
		log.Println("最后处理UTXO: ", tUnit.Hash.String())
		if stableCommissions, valid, err := tran.StableTx(tUnit); err != nil {
			log.Println(err)
			return
		} else {
			if valid {
				validUnits = append(validUnits, tUnit)
			} else {
				log.Println("该单元为无效单元: ", tUnit.Hash.String())
				tUnit.Invalid = true
				tran.db.SaveUnitToDb(tUnit)
			}
			allCommissions = append(allCommissions, stableCommissions...)
		}
	}

	if len(validUnits) > 0 {
		witnessCommission := tran.HandleCommission(validUnits)
		allCommissions = append(allCommissions, witnessCommission...)
	}

	authorsCom := make(map[common.Address][]types.Commission, 0)
	for _, com := range allCommissions {
		authorsCom[com.Address] = append(authorsCom[com.Address], com)
	}

	mc.distributeCommission(tran.db, authorsCom)
	//mainChain.ExtendStableUnit(units)
}

func (mc *MainChain) distributeCommission(db *boydb.DatabaseManager, comMap map[common.Address][]types.Commission) {
	for _, commissions := range comMap {
		db.SaveBatchUnspentOutput(commissions)
	}
}
