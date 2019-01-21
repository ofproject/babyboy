package dag

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/ds"
	"github.com/babyboy/common/queue"
	"github.com/babyboy/config"
	"github.com/babyboy/core/types"
	"log"
	"sort"
	"sync"
	"time"
	"babyboy-dag/boydb"
)

type MainChainUpdater struct {
	db          *leveldb.DatabaseManager
	unitHash    common.Hash
	parentList  []common.Hash
	stableHash  common.Hash
	stableLevel int64
	subHash     common.Hash
	mux         sync.RWMutex
}

func NewMainChainUpdater(db *boydb.DatabaseManager, unitHash common.Hash, parentList []common.Hash) *MainChainUpdater {
	mcu := MainChainUpdater{db: db, unitHash: unitHash, parentList: parentList}
	mcu.GetStableHashAndSubHash()
	mcu.mux = sync.RWMutex{}
	return &mcu
}

func (mcu *MainChainUpdater) GetStableHash(preUnitHash common.Hash) {

	unit, _ := mcu.db.GetUnitByHash(preUnitHash)

	if unit.IsStable && unit.IsOnMainChain {
		mcu.stableHash = unit.Hash
		mcu.stableLevel = unit.Level
		mcu.subHash = preUnitHash
		return
	}

	mcu.GetStableHash(unit.BestParentUnit)
}

func (mcu MainChainUpdater) GetMaxSubUnitLevelAtStableUnit() int64 {

	stableUnit, _ := mcu.db.GetUnitByHash(mcu.stableHash)
	maxLevel := int64(-1)
	for _, val := range mcu.parentList {
		tUnit, _ := mcu.db.GetUnitByHash(val)
		if tUnit.IsStable {
			continue
		}
		if tMcu.stableHash == mcu.stableHash && tMcu.subHash != mcu.subHash {
			tSubUnit, _ := mcu.db.GetUnitByHash(tMcu.subHash)
			if tSubUnit.WitnessedLevel > stableUnit.WitnessedLevel && tSubUnit.Level > maxLevel {
				maxLevel = tSubUnit.Level
			}
		}
	}
	return maxLevel
}

func (mcu *MainChainUpdater) ExtendStableUnit(units types.Units) {
	mcu.mux.Lock()
	defer mcu.mux.Unlock()
	currentRound := mcu.db.GetVoteRound()
	wr := NewWitnessReplacer()
	witnessSet := ds.NewAddressSet()
	witnessList := wr.wdb.GetWitnessesAsHash()
	witnessSet.ListInsert(witnessList)
	var newCampaigners []common.Address

	hashArray := types.NewHashArray()
	balls := make(types.Balls, 0)

	currentResult, _ := mcu.db.GetVoteResult(currentRound)
	tempTime := currentResult.EndTime
	if tempTime == 0 {
		currentResult.EndTime = time.Now().Unix()
		tempTime = currentResult.EndTime
		mcu.db.SaveVoteResult(currentRound, currentResult)
	}
	// Notice that the loop in this contract runs over an array which can be artificially inflated.
	for _, val := range units {
		ball := types.NewBall(mcu.subHash, val.ParentList, false)
		balls = append(balls, ball)
		hashArray.Hashes = append(hashArray.Hashes, val.Hash)
		if !val.Invalid {
			mcu.db.SaveTransactionAmount(val.Authors[0].Address, currentRound+1)
		}
		// todo use time to limit
		if val.TimeStamp >= tempTime+(config.MinIntervalTime-1)*3600 {
			amount, _ := mcu.db.GetTransactionAmount(val.Authors[0].Address, currentRound+1)
			if amount == config.MinTradeRate && !witnessSet.Exists(val.Authors[0].Address) {
				//mcu.db.SaveCandidateList(val.Authors[0].Address, currentRound+1)
				newCampaigners = append(newCampaigners, val.Authors[0].Address)
			}
		}

		// todo save temporarily

	}
}

func (mcu *MainChainUpdater) GetLastStableUnit() types.Unit {
	unit, err := mcu.db.GetUnitByHash(mcu.stableHash)
	if err != nil {
		return types.NewEmptyUnit()
	}
	return unit
}
