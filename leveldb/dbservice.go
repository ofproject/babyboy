package leveldb

import (
	"encoding/json"
	"log"
	"strings"
	"sync"

	"github.com/babyboy/common"
	"github.com/babyboy/config"
	"github.com/babyboy/core/types"
	"strconv"
)

var DbMgr *DatabaseManager
var once sync.Once

func GetDbInstance() *DatabaseManager {
	once.Do(func() {
		if DbMgr == nil {
			DbMgr = &DatabaseManager{}
		}
	})
	return DbMgr
}

type DatabaseManager struct {
	config.DataBaseConfig
	db *LDBDatabase
}

// Init DataBase
func (dbm *DatabaseManager) InitDatabase(dbPath string) error {
	var err error
	if dbm.db, err = dbm.OpenDatabase(dbPath, 0, 0); err != nil {
		log.Fatal(err)
		return err
	}

	return nil
}

// OpenDatabase
func (dbm *DatabaseManager) OpenDatabase(path string, cache int, handles int) (*LDBDatabase, error) {
	// Zhangxuesong TODO 检测一下 DataDir
	db, err := NewLDBDatabase(path, cache, handles)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func (dbm *DatabaseManager) Delete(key string) {
	err := dbm.db.Delete([]byte(key))
	if err != nil {
		log.Println(err)
	}
}

func (dbm *DatabaseManager) CloseDb() {
	dbm.db.Close()
}

// 存储创世单元
func (dbm *DatabaseManager) SaveGenisisUnit(unit interface{}) error {
	batch := dbm.db.NewBatch()
	jsonUnit, err := json.Marshal(unit)
	if err != nil {
		log.Fatalln(err)
		return err
	}
	keyUnit := strings.Join([]string{"unit.", config.GENISIS_UNIT_HASH}, "")
	batch.Put([]byte(keyUnit), jsonUnit)
	batch.Write()

	return nil
}

// 存储单元
func (dbm *DatabaseManager) SaveUnitToDb(unit types.Unit) {
	batch := dbm.db.NewBatch()

	keyUnit := strings.Join([]string{config.ConstDBUnitPrefix, unit.Hash.String()}, "")
	batch.Put([]byte(keyUnit), types.Unit2Byte(unit))
	batch.Write()
}

// 从数据库中删除单元
func (dbm *DatabaseManager) DelUnitFromDb(unit types.Unit) {
	batch := dbm.db.NewBatch()

	keyUnit := strings.Join([]string{config.ConstDBUnitPrefix, unit.Hash.String()}, "")
	batch.Delete([]byte(keyUnit))
	batch.Write()
}

// 存储多个单元
func (dbm *DatabaseManager) SaveUnitsToDb(units types.Units) {
	batch := dbm.db.NewBatch()
	for _, unit := range units {
		keyUnit := strings.Join([]string{config.ConstDBUnitPrefix, unit.Hash.String()}, "")
		batch.Put([]byte(keyUnit), types.Unit2Byte(unit))
	}
	batch.Write()
}

// 获取单元
func (dbm *DatabaseManager) GetUnitByHash(unitHash common.Hash) (types.Unit, error) {
	keyUnit := strings.Join([]string{config.ConstDBUnitPrefix, unitHash.String()}, "")
	readData, err := dbm.db.Get([]byte(keyUnit))
	unit := types.Byte2Unit(readData)

	return unit, err
}

// 在pending池中是否存在一笔UTXO
func (dbm *DatabaseManager) IsExistUnit(unitHash common.Hash) bool {
	unSpentKey := strings.Join([]string{config.ConstDBUnitPrefix, unitHash.String()}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(unSpentKey))
	isExist := it.Seek([]byte(""))

	return isExist
}

// 存储球
func (dbm *DatabaseManager) SaveBallToDb(ball types.Ball) {
	batch := dbm.db.NewBatch()

	keyBall := strings.Join([]string{config.ConstDBBallPrefix, ball.StringKey()}, "")
	batch.Put([]byte(keyBall), types.Ball2Byte(ball))
	batch.Write()
}

// 存储多个球
func (dbm *DatabaseManager) SaveBallsToDb(balls types.Balls) {
	batch := dbm.db.NewBatch()
	for _, ball := range balls {
		keyBall := strings.Join([]string{config.ConstDBBallPrefix, ball.StringKey()}, "")
		batch.Put([]byte(keyBall), types.Ball2Byte(ball))
	}
	batch.Write()
}

// 获取球
func (dbm *DatabaseManager) GetBallByHash(ballHash common.Hash) (types.Ball, error) {
	keyBall := strings.Join([]string{config.ConstDBBallPrefix, ballHash.String()}, "")
	readData, err := dbm.db.Get([]byte(keyBall))
	ball := types.Byte2Ball(readData)

	return ball, err
}

func (dbm *DatabaseManager) GetAllBalls() (types.Balls, error) {
	var allBalls types.Balls
	key := strings.Join([]string{config.ConstDBBallPrefix}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		log.Println(string(it.Key()))
		var msg types.Ball
		json.Unmarshal(it.Value(), &msg)
		allBalls = append(allBalls, msg)
		it.Next()
	}

	return allBalls, nil
}

// 存储单元列表
func (dbm *DatabaseManager) SaveParentsList(parentsList []common.Hash) {
	batch := dbm.db.NewBatch()

	for _, val := range parentsList {
		key := strings.Join([]string{config.ConstDBParentListPrefix, val.String()}, "")
		batch.Put([]byte(key), val.Bytes())
	}
	batch.Write()
}

// 获取单元列表
func (dbm *DatabaseManager) GetParentsList() []common.Hash {
	parentsList := make([]common.Hash, 0)
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBParentListPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		value := it.Value()
		parentsList = append(parentsList, common.BytesToHash(value))
		it.Next()
	}
	return parentsList
}

// 单独添加一个父单元
func (dbm *DatabaseManager) SaveParent(hash common.Hash) {
	batch := dbm.db.NewBatch()

	key := strings.Join([]string{config.ConstDBParentListPrefix, hash.String()}, "")
	batch.Put([]byte(key), hash.Bytes())

	batch.Write()
}

// 删除指定父单元
func (dbm *DatabaseManager) DelParent(hash common.Hash) {
	parent := strings.Join([]string{config.ConstDBParentListPrefix, hash.String()}, "")
	dbm.Delete(parent)
}

// 存储见证人列表
func (dbm *DatabaseManager) SaveWitnessList(witnesses []common.Address) {
	batch := dbm.db.NewBatch()
	for _, witness := range witnesses {
		wit := strings.Join([]string{config.ConstDBWitnessListPrefix, witness.String()}, "")
		batch.Put([]byte(wit), witness.Bytes())
	}
	batch.Write()
}

// 获取见证人列表
func (dbm *DatabaseManager) GetWitnessList() []common.Address {
	witnessList := make([]common.Address, 0)
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBWitnessListPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		value := it.Value()
		witnessList = append(witnessList, common.BytesToAddress(value))
		it.Next()
	}
	return witnessList
}

// 单独添加一个见证人
func (dbm *DatabaseManager) SaveWitness(hash common.Address) {
	batch := dbm.db.NewBatch()

	key := strings.Join([]string{config.ConstDBWitnessListPrefix, hash.String()}, "")
	batch.Put([]byte(key), hash.Bytes())

	batch.Write()
}

// 删除指定见证人
func (dbm *DatabaseManager) DelWitness(hash common.Address) {
	witness := strings.Join([]string{config.ConstDBWitnessListPrefix, hash.String()}, "")
	dbm.Delete(witness)
}

// 存储球数组
func (dbm *DatabaseManager) SaveBalls(balls []common.Hash) {
	batch := dbm.db.NewBatch()
	for _, witness := range balls {
		wit := strings.Join([]string{config.ConstDBBallPrefix, witness.String()}, "")
		batch.Put([]byte(wit), witness.Bytes())
		//log.Println(witness.String())
	}
	batch.Write()
}

// 获取所有球
func (dbm *DatabaseManager) GetBalls() []common.Hash {
	balls := make([]common.Hash, 0)
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBBallPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		value := it.Value()
		log.Println(common.BytesToHash(value).String())
		balls = append(balls, common.BytesToHash(value))
		it.Next()
	}
	return balls
}

// 删除指定球
func (dbm *DatabaseManager) DelBall(hash common.Hash) {
	witness := strings.Join([]string{config.ConstDBBallPrefix, hash.String()}, "")
	dbm.Delete(witness)
}

// ===================== UTXO 相关 ================
// 存入一笔未花费的output
func (dbm *DatabaseManager) SaveUnspentOutput(address common.Address, utxo types.UTXO) {
	batch := dbm.db.NewBatch()
	key := strings.Join([]string{config.ConstDBOutputPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	jsonStr, err := json.Marshal(utxo)
	if err != nil {
		log.Println(err)
		return
	}
	batch.Put([]byte(key), jsonStr)
	batch.Write()
}

// 删除一笔未花费的output
func (dbm *DatabaseManager) DelUnspentOutput(address common.Address, utxo types.UTXO) {
	batch := dbm.db.NewBatch()
	key := strings.Join([]string{config.ConstDBOutputPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	err := batch.Delete([]byte(key))
	if err != nil {
		log.Println("DelUnSpent Error ", err)
	}
	if err := batch.Write(); err != nil {
		log.Println(err)
	}

	//log.Println("Delete Stable UTXO")
	//strByte, _ := json.Marshal(utxo)
	//log.Println(string(strByte))
}

// 存入多笔未花费的output
func (dbm *DatabaseManager) SaveBatchUnspentOutput(commissions []types.Commission) {
	batch := dbm.db.NewBatch()
	for _, com := range commissions {
		key := strings.Join([]string{config.ConstDBOutputPrefix, com.Address.String(), ".", com.UTXO.ToHash().String()}, "")
		jsonStr, err := json.Marshal(com.UTXO)
		if err != nil {
			log.Println(err)
			return
		}
		batch.Put([]byte(key), jsonStr)

		//log.Println("Save Stable UTXO")
		//strByte, _ := json.Marshal(com.UTXO)
		//log.Println(string(strByte))
	}

	if err := batch.Write(); err != nil {
		log.Println(err)
	}
}

// 删除一笔pending池中的未花费的output
func (dbm *DatabaseManager) DelPendingUnspentOutput(address common.Address, utxo types.UTXO) {
	batch := dbm.db.NewBatch()
	unSpentKey := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	err := batch.Delete([]byte(unSpentKey))
	if err != nil {
		log.Println("Pending Unspent Error ", err)
	}
	if err := batch.Write(); err != nil {
		log.Println(err)
	}

	//log.Println("Delete Pending UTXO")
	//strByte, _ := json.Marshal(utxo)
	//log.Println(string(strByte))
}

// 获取指定地址所有未花费的unSpent
func (dbm *DatabaseManager) GetUnspentOutput(address common.Address) []types.UTXO {
	var utxos []types.UTXO
	key := strings.Join([]string{config.ConstDBOutputPrefix, address.String()}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		var msg types.UTXO
		json.Unmarshal(it.Value(), &msg)
		utxos = append(utxos, msg)
		it.Next()
	}

	return utxos
}

// 获取指定地址所有未花费的unSpent
func (dbm *DatabaseManager) IsExistUnspentOutput(address common.Address, utxo types.UTXO) bool {
	key := strings.Join([]string{config.ConstDBOutputPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	find := it.Seek([]byte(""))

	return find
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++
// 存入一笔UTXO到Pending池中
func (dbm *DatabaseManager) SavePendingUTXO(address common.Address, utxo types.UTXO) {
	batch := dbm.db.NewBatch()
	unSpentKey := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	//log.Println("WP: ", unSpentKey, ": ", utxo.Amount)
	jsonStr, err := json.Marshal(utxo)
	if err != nil {
		log.Println(err)
		return
	}
	err = batch.Put([]byte(unSpentKey), jsonStr)
	if err != nil {
		log.Println("PendingUnit Error ", err)
	}
	batch.Write()
}

// 删除一笔UTXO从Pending池中
func (dbm *DatabaseManager) DelPendingUTXO(address common.Address, utxo types.UTXO) {
	batch := dbm.db.NewBatch()
	unSpentKey := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	//log.Println("DP: ", unSpentKey, ": ", utxo.Amount)
	err := batch.Delete([]byte(unSpentKey))
	if err != nil {
		log.Println("PendingUnit Error ", err)
	}
	batch.Write()
}

// 查找一笔UTXO是否存在于Pending池中
func (dbm *DatabaseManager) GetPendingUTXO(address common.Address, hash common.Hash) types.UTXO {
	utxo := types.NewEmptyUTXO()
	unSpentKey := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String(), ".", hash.String()}, "")
	//log.Println("RP: ", unSpentKey)
	it := dbm.db.NewIteratorWithPrefix([]byte(unSpentKey))
	it.Seek([]byte(""))
	for it.Valid() {
		var msg types.UTXO
		json.Unmarshal(it.Value(), &msg)
		utxo = msg
		it.Next()
	}

	return utxo
}

// 查找一笔UTXO是否存在于Pending池中通过指定作者
func (dbm *DatabaseManager) GetPendingUTXOByAuthor(address common.Address) []types.UTXO {
	var pendingUnspent []types.UTXO
	unSpentKey := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String()}, "")
	//log.Println("RP: ", unSpentKey)
	it := dbm.db.NewIteratorWithPrefix([]byte(unSpentKey))
	it.Seek([]byte(""))
	for it.Valid() {
		var msg types.UTXO
		json.Unmarshal(it.Value(), &msg)
		pendingUnspent = append(pendingUnspent, msg)
		it.Next()
	}

	return pendingUnspent
}

// 在pending池中是否存在一笔UTXO
func (dbm *DatabaseManager) IsExistPendingUTXO(address common.Address, utxo types.UTXO) bool {
	unSpentKey := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String(), ".", utxo.ToHash().String()}, "")
	//log.Println("RP: ", unSpentKey)
	it := dbm.db.NewIteratorWithPrefix([]byte(unSpentKey))
	isExist := it.Seek([]byte(""))

	return isExist
}

// 获取指定地址所有Pending未花费
func (dbm *DatabaseManager) GetAllPendingUnSpent(address common.Address) []types.UTXO {
	var spent []types.UTXO
	key := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String(), "."}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		log.Println(string(it.Key()))
		var msg types.UTXO
		json.Unmarshal(it.Value(), &msg)
		spent = append(spent, msg)
		it.Next()
	}

	return spent
}

// +++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++

// 获取指定地址所有未花费的unSpent从Pending池中
func (dbm *DatabaseManager) GetUnspentOutputFromPendingPool(address common.Address) map[string]types.UTXO {
	var unSpent = make(map[string]types.UTXO, 1)
	key := strings.Join([]string{config.ConstDBPendingUnitPrefix, address.String()}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		var msg types.UTXO
		json.Unmarshal(it.Value(), &msg)
		unSpent[string(it.Key())] = msg
		it.Next()
	}

	return unSpent
}

// 获取所有Pending状态的UTXO
func (dbm *DatabaseManager) GetAllPendingOutput() {
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBPendingUnitPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		key := it.Key()
		value := it.Value()

		log.Println(string(key))
		log.Println(string(value))
		it.Next()
	}
}

// 获取所有Stable状态的UTXO
func (dbm *DatabaseManager) GetAllStableOutput() {
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBOutputPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		key := it.Key()
		value := it.Value()

		log.Println(string(key))
		log.Println(string(value))
		it.Next()
	}
}

// 返回所有单元
func (dbm *DatabaseManager) GetAllUnits(fn func(hash string, value string)) {
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBUnitPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		key := it.Key()
		if fn != nil {
			fn(string(key), string(it.Value()))
		}
		it.Next()
	}
}

// 获取所有单元的总数量
func (dbm *DatabaseManager) GetAllUnitCount() int64 {
	var count int64
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBUnitPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		count++
		it.Next()
	}

	return count
}

// 获取稳定点数量
func (dbm *DatabaseManager) GetAllStableUnitCount() int64 {
	var count int64
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBUnitPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		var unit types.Unit
		json.Unmarshal(it.Value(), &unit)
		if unit.IsStable {
			count++
		}
		it.Next()
	}

	return count
}

// 获取不稳定点数量
func (dbm *DatabaseManager) GetAllUnStableUnitCount() int64 {
	var count int64
	it := dbm.db.NewIteratorWithPrefix([]byte(config.ConstDBUnitPrefix))
	it.Seek([]byte(""))
	for it.Valid() {
		var unit types.Unit
		json.Unmarshal(it.Value(), &unit)
		if !unit.IsStable {
			count++
		}
		it.Next()
	}

	return count
}

// 存入单元的子单元
func (dbm *DatabaseManager) SaveChildrenUnit(parentunit common.Hash, children types.HashArray) {
	batch := dbm.db.NewBatch()

	key := strings.Join([]string{config.ConstDBChildrenHash, parentunit.String()}, "")
	log.Println("Save Children: ", key, children)
	childrenByte, err := json.Marshal(children)
	if err != nil {
		log.Println(err)
		return
	}
	err = batch.Put([]byte(key), childrenByte)
	if err != nil {
		log.Println("Save Children Error ", err)
	}

	batch.Write()
}

// 获得单元的子单元
func (dbm *DatabaseManager) GetChildrenUnit(hash common.Hash) (types.HashArray, error) {
	children := types.NewHashArray()
	key := strings.Join([]string{config.ConstDBChildrenHash, hash.String()}, "")
	//log.Println("Get Children: ", key)
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		json.Unmarshal(it.Value(), &children)
		it.Next()
	}

	return children, nil
}

// 存入主链上稳定点对应稳定的单元列表
func (dbm *DatabaseManager) SaveStableUnits(mcUnit common.Hash, stableUnits types.HashArray) {
	batch := dbm.db.NewBatch()

	key := strings.Join([]string{config.ConstDBStableUnitsPrefix, mcUnit.String()}, "")
	//log.Println("Save Stable Units: ", key, stableUnits)
	stableUnitsByte, err := json.Marshal(stableUnits)
	if err != nil {
		log.Println(err)
		return
	}
	if err := batch.Put([]byte(key), stableUnitsByte); err != nil {
		log.Println("Save Stable Units Error ", err)
		return
	}
	batch.Write()
}

// 获得主链上稳定点对应稳定的单元列表
func (dbm *DatabaseManager) GetStableUnits(hash common.Hash) (types.Units, error) {
	stableUnitsHashes := types.NewHashArray()
	key := strings.Join([]string{config.ConstDBStableUnitsPrefix, hash.String()}, "")
	log.Println("Get Stable Units: ", key)
	stableUnits := types.NewUnits()
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		json.Unmarshal(it.Value(), &stableUnitsHashes)
		it.Next()
	}
	for _, val := range stableUnitsHashes.Hashes {
		unit, err := dbm.GetUnitByHash(val)
		if err != nil {
			log.Panicln("Get Stable Unit Error:", err)
			return types.Units{}, err
		}
		stableUnits = append(stableUnits, unit)
	}
	return stableUnits, nil
}

// 存入当前收到的最新的投票轮数
func (dbm *DatabaseManager) SaveVoteRound(voteRound int64) {
	batch := dbm.db.NewBatch()
	key := strings.Join([]string{config.ConstDBVoteRound}, "")
	voteRoundByte, err := json.Marshal(voteRound)
	if err != nil {
		log.Println(err)
		return
	}
	err = batch.Put([]byte(key), voteRoundByte)
	if err != nil {
		log.Println("Save VoteRound Error ", err)
	}

	batch.Write()
}

// 获得当前收到的最新的投票轮数
func (dbm *DatabaseManager) GetVoteRound() (int64, error) {
	var voteRound int64
	key := strings.Join([]string{config.ConstDBVoteRound}, "")
	//log.Println("Get Children: ", key)
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		json.Unmarshal(it.Value(), &voteRound)
		it.Next()
	}

	return voteRound, nil
}

// 存入某一轮投票结果
func (dbm *DatabaseManager) SaveVoteResult(voteRound int64, result types.VoteResult) {
	batch := dbm.db.NewBatch()

	key := strings.Join([]string{config.ConstDBVoteResult, strconv.FormatInt(voteRound, 10)}, "")
	resultByte, err := json.Marshal(result)
	if err != nil {
		log.Println(err)
		return
	}
	err = batch.Put([]byte(key), resultByte)
	if err != nil {
		log.Println("Save vote result Error ", err)
	}

	batch.Write()
}

// 获得某一轮投票结果
func (dbm *DatabaseManager) GetVoteResult(voteRound int64) (types.VoteResult, error) {
	var voteResult types.VoteResult
	key := strings.Join([]string{config.ConstDBVoteResult, strconv.FormatInt(voteRound, 10)}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		json.Unmarshal(it.Value(), &voteResult)
		it.Next()
	}

	return voteResult, nil
}

// 清空数据
func (dbm *DatabaseManager) DelAllData(key string) {
	batch := dbm.db.NewBatch()
	unSpentKey := strings.Join([]string{config.ConstDBUnitPrefix, key}, "")
	err := batch.Delete([]byte(unSpentKey))
	if err != nil {
		log.Println("Delete Error ", err)
	}
	batch.Write()
}

// 缓存未发送的单元
func (dbm *DatabaseManager) SaveCacheUnitToDb(unit types.Unit) {
	batch := dbm.db.NewBatch()

	keyUnit := strings.Join([]string{config.ConstCacheUnit, unit.Hash.String()}, "")
	batch.Put([]byte(keyUnit), types.Unit2Byte(unit))
	batch.Write()
}

// 缓存未发送的单元
func (dbm *DatabaseManager) GetCacheUnitFromDb() types.Units {
	cacheUnits := types.Units{}
	key := strings.Join([]string{config.ConstCacheUnit}, "")
	it := dbm.db.NewIteratorWithPrefix([]byte(key))
	it.Seek([]byte(""))
	for it.Valid() {
		unit := types.Unit{}
		json.Unmarshal(it.Value(), &unit)
		cacheUnits = append(cacheUnits, unit)
		it.Next()
	}

	return cacheUnits
}

// 清除缓存未发送的单元
func (dbm *DatabaseManager) DelCacheUnitFromDb(unit types.Unit) {
	batch := dbm.db.NewBatch()
	key := strings.Join([]string{config.ConstCacheUnit, unit.Hash.String()}, "")
	err := batch.Delete([]byte(key))
	if err != nil {
		log.Println("Delete Error ", err)
	}
	batch.Write()
}
