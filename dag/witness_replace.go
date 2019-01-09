package dag

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/ds"
	"github.com/babyboy/common/queue"
	"github.com/babyboy/core/types"
	"github.com/babyboy/dag/memdb"
	"babyboy-dag/boydb"
)

type WitnessReplacer struct {
	db             *leveldb.DatabaseManager
	wdb            *memdb.WitnessMemDB
	pdb            *memdb.ParentMemDB
	StartTime      int64
	EndTime        int64
	round          int64
	voteResult     common.Address
	replaceWitness common.Address
	parentList     []common.Hash
}

func NewWitnessReplacer(timeStamp int64, round int64, voteResult common.Address, replaceWitness common.Address) *WitnessReplacer {
	db := leveldb.GetDbInstance()
	wdb := memdb.GetWitnessMemDBInstance()
	pdb := memdb.GetParentMemDBInstance()
	voteRound := wdb.GetVoteRound()
	startTime := wdb.GetVoteResultByRound(voteRound).EndTime
	endTime := timeStamp
	parentList := pdb.GetParentsAsHash()
	return &WitnessReplacer{db, wdb, pdb, startTime, endTime, round, voteResult, replaceWitness, parentList}
}

func (wr *WitnessReplacer) GetReplacedWitness() (common.Address, int64) {
	que := queue.New()
	for _, parentHash := range wr.parentList {
		tUnit, _ := wr.db.GetUnitByHash(parentHash)
		if tUnit.TimeStamp <= wr.EndTime && tUnit.TimeStamp >= wr.StartTime {
			que.Push(parentHash)
		}
	}

	witnessSet := ds.NewAddressSet()
	witnessList := wr.wdb.GetWitnessesAsHash()
	witnessSet.ListInsert(witnessList)
	mp := make(map[common.Address]int64, 0)
	var minTimes int64
	var minWitness common.Address
	minTimes = 0

	for !que.Empty() {
		hash := que.Front().(common.Hash)
		tUnit, _ := wr.db.GetUnitByHash(hash)
		if witnessSet.Exists(tUnit.Authors[0].Address) {
			mp[tUnit.Authors[0].Address] += 1
		}
		for _, val := range tUnit.ParentList {
			tUnit, _ := wr.db.GetUnitByHash(val)
			if tUnit.TimeStamp <= wr.EndTime && tUnit.TimeStamp >= wr.StartTime {
				que.Push(val)
			}
		}
	}

	for _, witness := range witnessList {
		if mp[witness] < minTimes {
			minTimes = mp[witness]
			minWitness = witness
		}
	}
	return minWitness, minTimes

}

func (wr *WitnessReplacer) GetCampaignList(minTimes int64) []common.Address {

	// 父节点列表的单元插入队列
	que := queue.New()
	for _, parentHash := range wr.parentList {
		tUnit, _ := wr.db.GetUnitByHash(parentHash)
		if tUnit.TimeStamp <= wr.EndTime && tUnit.TimeStamp >= wr.StartTime {
			que.Push(parentHash)
		}
	}

	witnessSet := ds.NewAddressSet()
	witnessList := wr.wdb.GetWitnessesAsHash()
	witnessSet.ListInsert(witnessList)
	mp := make(map[common.Address]int64, 0)

	for !que.Empty() {
		hash := que.Front().(common.Hash)
		tUnit, _ := wr.db.GetUnitByHash(hash)
		if !witnessSet.Exists(tUnit.Authors[0].Address) {
			mp[tUnit.Authors[0].Address] += 1
		}
		for _, val := range tUnit.ParentList {
			tUnit, _ := wr.db.GetUnitByHash(val)
			if tUnit.TimeStamp <= wr.EndTime && tUnit.TimeStamp >= wr.StartTime {
				que.Push(val)
			}
		}
	}

	campaignList := make([]common.Address, 0)
	for key, val := range mp {
		if val > minTimes {
			campaignList = append(campaignList, key)
		}
	}
	return campaignList
}

func (wr *WitnessReplacer) SaveVoteResult() {
	result := types.NewVoteResult(wr.StartTime, wr.EndTime, wr.voteResult, wr.replaceWitness, wr.round)
	wr.wdb.SaveVoteRound(wr.round)
	wr.wdb.SaveVoteResultByRound(wr.round, result)
}
