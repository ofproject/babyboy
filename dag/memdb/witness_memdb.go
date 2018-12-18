package memdb

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/ds"
	"github.com/babyboy/config"
	"github.com/babyboy/core/types"
	"log"
	"sync"
)

var WmDb *WitnessMemDB
var onceWmDb sync.Once

// 获取见证人存储实例
func GetWitnessMemDBInstance() *WitnessMemDB {
	onceWmDb.Do(func() {
		if WmDb == nil {
			WmDb = NewWitnessMemDB()
		}
	})
	return WmDb
}

type WitnessMemDB struct {
	db           *boydb.DatabaseManager     //数据库
	witnessSet   *ds.AddressSet             //见证人集合
	replaceRound int64                      //当前的替换完成的轮数
	voteRound    int64                      //已存储的所有轮数
	replaceMap   map[int64]types.VoteResult //替换的见证人结果的集合，只记录还未替换到的剩余几轮结果
}

// 新建一个WitnessMemDB
func NewWitnessMemDB() *WitnessMemDB {
	db := &boydb.DatabaseManager{}
	witnessSet := ds.NewAddressSet()
	return &WitnessMemDB{db, witnessSet, -1, -1, map[int64]types.VoteResult{}}
}

// 初始化WitnessMemDB
func (wdb *WitnessMemDB) InitWitnessMemDB(db *boydb.DatabaseManager) {
	wdb.db = db
	wdb.witnessSet.ListInsert(db.GetWitnessList())
	wdb.voteRound, _ = db.GetVoteRound()
	wdb.replaceRound = wdb.voteRound - config.Const_Stable_Rounds
	for i := wdb.replaceRound - config.Const_Stable_Rounds; i < wdb.replaceRound; i++ {
		wdb.replaceMap[i], _ = db.GetVoteResult(i)
	}
}

// ***见证人存储相关函数*** //

// 保存见证人
func (wdb *WitnessMemDB) SaveWitness(witness common.Address) {
	wdb.db.SaveWitness(witness)
	wdb.witnessSet.Insert(witness)
}

// 保存见证人列表
func (wdb *WitnessMemDB) SaveWitnessList(witnessList []common.Address) {
	wdb.db.SaveWitnessList(witnessList)
	wdb.witnessSet.ListInsert(witnessList)
}

// 删除见证人
func (wdb *WitnessMemDB) DeleteWitness(witness common.Address) {
	wdb.db.DelWitness(witness)
	wdb.witnessSet.Remove(witness)
}

// 替换见证人
func (wdb *WitnessMemDB) ReplaceWitness(oldWitness, newWitness common.Address) {
	if wdb.witnessSet.Exists(oldWitness) {
		wdb.DeleteWitness(oldWitness)
		wdb.SaveWitness(oldWitness)
	}
}

// 获取见证人列表，哈希数组
func (wdb *WitnessMemDB) GetWitnessesAsHash() []common.Address {
	if wdb.witnessSet.Empty() {
		wdb.witnessSet.ListInsert(wdb.db.GetWitnessList())
	}
	return wdb.witnessSet.GetAllAddress()
}

// 获取见证人列表，字符串数组
func (wdb *WitnessMemDB) GetWitnessesAsString() []string {
	if wdb.witnessSet.Empty() {
		wdb.witnessSet.ListInsert(wdb.db.GetWitnessList())
	}
	return wdb.witnessSet.GetAllAddressAsString()
}

// 存储最新的投票轮数
func (wdb *WitnessMemDB) SaveVoteRound(round int64) {
	wdb.voteRound = round
	wdb.db.SaveVoteRound(round)
	wdb.replaceRound = wdb.voteRound - config.Const_Stable_Rounds
}

// 获取最新的投票轮数
func (wdb *WitnessMemDB) GetVoteRound() int64 {
	return wdb.voteRound
}

// 存储见证人替换结果
func (wdb *WitnessMemDB) SaveVoteResultByRound(round int64, result types.VoteResult) {
	wdb.replaceMap[round] = result
	wdb.db.SaveVoteResult(round, result)

}

// 获取见证人替换结果
func (wdb *WitnessMemDB) GetVoteResultByRound(round int64) types.VoteResult {

	_, ok := wdb.replaceMap[round]
	if ok {
		return wdb.replaceMap[round]
	} else {
		result, _ := wdb.db.GetVoteResult(round)
		return result
	}
}

// 存储见证人替换结果
func (wdb *WitnessMemDB) ReplaceWitnessByResult(result types.VoteResult) {
	wdb.replaceMap[result.Round] = result
	wdb.db.SaveVoteResult(result.Round, result)
	if result.Round-config.Const_Stable_Rounds-wdb.voteRound == 1 {
		wdb.ReplaceWitness(result.ReplacedWitness, result.VoteResult)
		wdb.voteRound = result.Round - config.Const_Stable_Rounds
	} else {
		log.Println("replace witness err!", "  now round:", wdb.voteRound, "replace round:", result.Round)
	}
}

// ******************* //
