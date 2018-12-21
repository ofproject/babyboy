package types

import (
	"encoding/json"
	"log"

	"github.com/babyboy/common"
	"github.com/babyboy/crypto/sha3"
	"github.com/babyboy/babyboy/rlp"
)

type Units []Unit

func NewUnits() Units {
	units := make(Units, 0)
	return units
}

func (units Units) Len() int {
	return len(units)
}

func (units Units) Swap(i, j int) {

	units[i], units[j] = units[j], units[i]
}

func (units Units) Less(i, j int) bool {
	return units[i].Level < units[j].Level
}

type Unit struct {
	Hash              common.Hash      `json:"hash"`
	Version           string           `json:"version"`
	WitnessList       []common.Address `json:"witness_list"`
	LastBallUnit      common.Hash      `json:"last_ball_unit"`
	HeadersCommission int              `json:"headers_commission"` // 分给子单元的交易手续费
	PayloadCommission int              `json:"payload_commission"` // 分给见证人的交易手续费
	TimeStamp         int64            `json:"timestamp"`
	ParentList        []common.Hash    `json:"parent_list"`
	Authors           Authors          `json:"authors"`
	Messages          Messages         `json:"messages"`

	BestParentUnit   common.Hash    `json:"best_parent_unit"`
	MainChainIndex   int64          `json:"main_chain_index"`
	IsStable         bool           `json:"is_stable"`
	IsOnMainChain    bool           `json:"is_on_main_chain"`
	Level            int64          `json:"level"`
	WitnessedLevel   int64          `json:"witnessed_level"`
	SubStableMinHash common.Hash    `json:"sub_stable_min_hash"`
	SubStableAuthor  common.Address `json:"sub_stable_author"`
	Invalid          bool           `json:"is_good"`
}

// 单元不修改的部分转换成hash用作数据库的Key值
func (u *Unit) HashKey() common.Hash {
	type UnitJSON struct {
		Version           string           `json:"version"`
		Messages          Messages         `json:"messages"`
		Authors           Authors          `json:"authors"`
		LastBallUnit      common.Hash      `json:"last_ball_unit"`
		ParentList        []common.Hash    `json:"parent_list"`
		WitnessList       []common.Address `json:"witness_list"`
		HeadersCommission int              `json:"headers_commission"` // 分给子单元的交易手续费
		PayloadCommission int              `json:"payload_commission"` // 分给见证人的交易手续费
	}
	var uJson UnitJSON
	uJson.Version = u.Version
	uJson.Messages = u.Messages
	uJson.Authors = u.Authors
	uJson.LastBallUnit = u.LastBallUnit
	uJson.ParentList = u.ParentList
	uJson.WitnessList = u.WitnessList
	uJson.HeadersCommission = u.HeadersCommission
	uJson.PayloadCommission = u.PayloadCommission
	jsonByte, _ := json.Marshal(uJson)
	hash := sha3.Sum256(jsonByte)
	return hash
}

// 单元不修改的部分转换成string用作数据库的Key值
func (u Unit) StringKey() string {
	hash := u.HashKey()
	s := hash.String()
	return s
}

// 单元修改为稳定
func (u *Unit) ChangeStable(mainChainIndex int64) {
	u.IsStable = true
	u.MainChainIndex = mainChainIndex
}

// 单元更新SubStableMinHash
func (u *Unit) UpdataSubStableMinHash(hash common.Hash, author common.Address) {
	if u.SubStableMinHash.String() > hash.String() {
		u.SubStableMinHash = hash
		u.SubStableAuthor = author
	}
}

// 单元内容重置
func (u *Unit) ResetStableState() {
	u.IsStable = false
	u.MainChainIndex = 0
	u.IsOnMainChain = false
}

// 输出计算出的信息
func (u Unit) Print() {
	log.Println("level:           ", u.Level)
	log.Println("witness level:   ", u.WitnessedLevel)
	log.Println("mainchain index: ", u.MainChainIndex)
}

// 修改后的单元序列化转换成string
func Unit2String(u Unit) string {
	jsonByte, _ := json.Marshal(u)
	return string(jsonByte)
}

// 修改后的单元序列化转换成[]byte
func Unit2Byte(u Unit) []byte {
	jsonByte, _ := json.Marshal(u)
	return jsonByte
}

// 字符串反序列化转换成Unit
func String2Unit(unit string) Unit {
	var u Unit
	err := json.Unmarshal([]byte(unit), &u)
	if err != nil {
		log.Println(err)
		return u
	}
	return u
}

// 字符串反序列化转换成Unit
func Byte2Unit(unit []byte) Unit {
	var u Unit
	json.Unmarshal(unit, &u)
	return u
}

// rlp hash
func RlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// Create new unit
func NewUnit(parentList []common.Hash, witnessList []common.Address, bestParentUnit common.Hash,
	lastBallUnit common.Hash, level int64, witnessLevel int64, witness int) Unit {
	var u Unit
	u.Version = "1.0"
	u.Messages = Messages{}
	u.Authors = Authors{}
	u.ParentList = parentList
	u.Level = level
	u.LastBallUnit = lastBallUnit
	u.WitnessList = witnessList
	u.HeadersCommission = 0
	u.PayloadCommission = 0
	u.BestParentUnit = bestParentUnit
	u.WitnessedLevel = 0
	u.WitnessedLevel = witnessLevel
	u.IsStable = false
	u.MainChainIndex = 0
	u.IsOnMainChain = false
	u.SubStableMinHash = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	return u
}

// New Empty unit
func NewEmptyUnit() Unit {
	return Unit{}
}

// CalculateHash hashes the values of a TestContent
func (u Unit) CalculateHash() ([]byte, error) {
	return u.HashKey().Bytes(), nil
}

// Equals tests for equality of two Contents
func (u Unit) Equals(other common.Content) (bool, error) {
	return u.Hash == other.(Unit).Hash, nil
}
