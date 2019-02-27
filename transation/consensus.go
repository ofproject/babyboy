package transaction

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/ds"
	"github.com/babyboy/core/types"
	"github.com/babyboy/dag"
	"github.com/babyboy/memdb"
	"errors"
	"log"
	"sync"
)

type MainChain struct {
	db    *boydb.DatabaseManager
	mutex sync.Mutex
}

func NewMainChain() *MainChain {
	mc := &MainChain{}
	mc.mutex = sync.Mutex{}

	return mc
}

func (mc *MainChain) UpdateMainChain(tran *Transaction, unit types.Unit) error {
	db := boydb.GetDbInstance()
	pdb := memdb.GetParentMemDBInstance()

	unit.ResetStableState()
	boydb.GetDbInstance().SaveUnitToDb(unit)
	pdb.SaveParent(unit.Hash)

	mainChain := dag.NewMainChainUpdater(db, unit.Hash, pdb.GetParentsAsHash())
	//start := time.Now()
	for {
		canExtend, units := mainChain.StableBallCanExtend()
		if !canExtend {
			break
		}
		//start := time.Now()
		mc.handleStableUnits(tran, db, units, mainChain)
	}

	return nil
}

func (mc *MainChain) ValidUnit(unit types.Unit) error {
	if !mc.checkUnitGraphInfo(unit) {
		return errors.New("单元验证失败")
	}

	return nil
}

func (mc *MainChain) handleStableUnits(tran *Transaction, db *boydb.DatabaseManager, units types.Units, mainChain *dag.MainChainUpdater) {
	//mc.mutex.Lock()
	//defer mc.mutex.Unlock()

	allCommissions := make([]types.Commission, 0)

	Len := len(units)

	validUnits := types.Units{}

	for i := 0; i < Len; i++ {
		tUnit := units[i]
		if stableCommissions, valid, err := tran.StableTx(tUnit); err != nil {
			log.Println(err)
			return
		} else {
			if valid {
				validUnits = append(validUnits, tUnit)
			}
			allCommissions = append(allCommissions, stableCommissions...)
		}
	}


	if len(validUnits) > 0 {
		witnessCommission:= tran.HandleCommission(validUnits)
		allCommissions = append(allCommissions, witnessCommission...)
	}

	authorsCom := make(map[common.Address][]types.Commission, 0)
	for _, com := range allCommissions {
		authorsCom[com.Address] = append(authorsCom[com.Address], com)
	}

	mc.distributeCommission(db, authorsCom)
	mainChain.ExtendStableUnit(units)
}

