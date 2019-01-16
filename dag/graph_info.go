package dag

import (
	"sort"

	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/ds"
	"github.com/babyboy/common/queue"
	"github.com/babyboy/config"
	"github.com/babyboy/core/types"
)

type GraphInfoGetter struct {
	db          *boydb.DatabaseManager
	parentList  []common.Hash
	witnessList []common.Address
	bestParent  common.Hash
}

func NewGraphInfoGetter(db *boydb.DatabaseManager, parentList []common.Hash, witnessList []common.Address) *GraphInfoGetter {
	gig := GraphInfoGetter{db: db, parentList: parentList, witnessList: witnessList}
	gig.GetBestParentUnit()
	return &gig
}

func (gig GraphInfoGetter) GetLevel() int64 {
	level := int64(0)

	for _, parentHash := range gig.parentList {
		unit, _ := gig.db.GetUnitByHash(parentHash)
		tLevel := unit.Level
		if tLevel > level {
			level = tLevel
		}
	}

	return int64(level + 1)
}

func (gig GraphInfoGetter) GetWitnessLevel() int64 {

	if len(gig.bestParent) == 0 {
		return -1
	}
	bestParent := gig.bestParent

	unit, _ := gig.db.GetUnitByHash(bestParent)

	stableWitnessSet := ds.NewAddressSet()
	stableWitnessSet.ListInsert(gig.witnessList)

	witnessCountSet := ds.NewAddressSet()
	witnessLevel := unit.Level
	que := queue.New()
	que.Push(bestParent)

	for !que.Empty() {
		unit, _ := gig.db.GetUnitByHash(que.Front().(common.Hash))
		que.Pop()
		if stableWitnessSet.Exists(unit.Authors[0].Address) {
			witnessCountSet.Insert(unit.Authors[0].Address)
			if unit.Level < witnessLevel {
				witnessLevel = unit.Level
			}
			if witnessCountSet.Size() == config.MajorityOfWitnesses {
				return witnessLevel
			}
		}
		if unit.Hash.String() == config.GENISIS_UNIT_HASH {
			return 0
		}
		que.Push(unit.BestParentUnit)
	}
	return -1
}

func (gig *GraphInfoGetter) GetBestParentUnit() common.Hash {

	if len(gig.parentList) == 0 {
		return common.Hash{}
	}
	bestParentUnit, _ := gig.db.GetUnitByHash(gig.parentList[0])

	for _, parentHash := range gig.parentList {
		tParentUnit, _ := gig.db.GetUnitByHash(parentHash)
		if bestParentUnit.WitnessedLevel < tParentUnit.WitnessedLevel {
			bestParentUnit = tParentUnit
			continue
		}
		if bestParentUnit.WitnessedLevel == tParentUnit.WitnessedLevel && bestParentUnit.Level > tParentUnit.Level {
			bestParentUnit = tParentUnit
			continue
		}
		if bestParentUnit.WitnessedLevel == tParentUnit.WitnessedLevel && bestParentUnit.Level == tParentUnit.Level && bestParentUnit.StringKey() > tParentUnit.StringKey() {
			bestParentUnit = tParentUnit
			continue
		}
	}
	gig.bestParent = bestParentUnit.Hash

	return gig.bestParent
}

func (gig GraphInfoGetter) GetLastStableBall() common.Hash {

	if len(gig.bestParent) == 0 {
		gig.GetBestParentUnit()
	}

	bestParent := gig.bestParent
	var unit types.Unit
	var subHash common.Hash

	que := queue.New()
	que.Push(bestParent)
	for !que.Empty() {
		subHash = que.Front().(common.Hash)
		que.Pop()
		unit, _ = gig.db.GetUnitByHash(subHash)
		if unit.IsStable && unit.IsOnMainChain {
			return unit.Hash
		}
		que.Push(unit.BestParentUnit)
	}
	return unit.Hash
}

func (gig GraphInfoGetter) GetLastStableBallMCI() int64 {

	if len(gig.bestParent) == 0 {
		gig.GetBestParentUnit()
	}

	bestParent := gig.bestParent
	var unit types.Unit
	var subHash common.Hash

	que := queue.New()
	que.Push(bestParent)
	for !que.Empty() {
		subHash = que.Front().(common.Hash)
		que.Pop()
		unit, _ = gig.db.GetUnitByHash(subHash)

		if unit.IsStable && unit.IsOnMainChain {
			return unit.MainChainIndex
		}
		que.Push(unit.BestParentUnit)
	}
	return unit.MainChainIndex
}

func (gig GraphInfoGetter) GetMissingUnits(lastStableMCI, lastKnownMCI int64) (types.Units, types.Units) {

	if lastKnownMCI > lastStableMCI {
		return nil, nil
	}

	stableUnitMap := ds.NewHashMap()
	unstableUnitMap := ds.NewHashMap()

	que := queue.New()
	var unit types.Unit

	for _, val := range gig.parentList {
		unit, _ = gig.db.GetUnitByHash(val)

		if !unstableUnitMap.Exists(val) {
			unstableUnitMap.Insert(val, unit)
			que.Push(val)

		}
	}

	for !que.Empty() {
		unit, _ = gig.db.GetUnitByHash(que.Front().(common.Hash))
		que.Pop()
		for _, val := range unit.ParentList {
			unit, _ = gig.db.GetUnitByHash(val)
			if unit.IsStable {
				if !stableUnitMap.Exists(val) && unit.MainChainIndex > lastKnownMCI {
					stableUnitMap.Insert(val, unit)
					que.Push(val)
				}
			} else {
				if !unstableUnitMap.Exists(val) {
					unstableUnitMap.Insert(val, unit)
					que.Push(val)

				}
			}
		}
	}

	stableUnits := stableUnitMap.GetAllValue()
	unstableUnits := unstableUnitMap.GetAllValue()

	if !sort.IsSorted(stableUnits) {
		sort.Sort(stableUnits)
	}
	if !sort.IsSorted(unstableUnits) {
		sort.Sort(unstableUnits)
	}
	return stableUnits, unstableUnits
}

func (gig GraphInfoGetter) GetMissingStableUnitsHashOnMainChain(lastStableMCI, lastKnownMCI int64) []common.Hash {

	if lastKnownMCI > lastStableMCI {
		return nil
	}

	stableBallMap := ds.NewHashMap()

	if len(gig.bestParent) == 0 {
		gig.GetBestParentUnit()
	}

	bestParent := gig.bestParent
	var unit types.Unit
	var hash common.Hash

	que := queue.New()
	que.Push(bestParent)
	for !que.Empty() {
		hash = que.Front().(common.Hash)
		que.Pop()
		unit, _ = gig.db.GetUnitByHash(hash)

		if unit.IsOnMainChain && unit.MainChainIndex >= lastKnownMCI && unit.MainChainIndex <= lastStableMCI {
			stableBallMap.Insert(hash, unit)
		}
		que.Push(unit.BestParentUnit)
	}
	stableBallUnits := stableBallMap.GetAllValue()
	if !sort.IsSorted(stableBallUnits) {
		sort.Sort(stableBallUnits)
	}

	stableBallUnitsHash := make([]common.Hash, 0)
	for _, val := range stableBallUnits {
		stableBallUnitsHash = append(stableBallUnitsHash, val.Hash)
	}

	return stableBallUnitsHash
}
