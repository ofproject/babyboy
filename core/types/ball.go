package types

import (
	"encoding/json"

	"github.com/babyboy/common"
)

type Balls []Ball

type Ball struct {
	UnitHash    common.Hash   `json:"unit_hash"`
	ParentBalls []common.Hash `json:"parent_balls"`
	IsInvalid   bool          `json:"is_invalid"`
}

func NewBall(unitHash common.Hash, parentBalls []common.Hash, isInvalid bool) Ball {
	var b Ball
	b.UnitHash = unitHash
	b.ParentBalls = parentBalls
	b.IsInvalid = isInvalid
	return b
}

func (b Ball) HashKey() common.Hash {
	return b.UnitHash
}

func (b Ball) StringKey() string {
	return b.HashKey().String()
}

func Ball2Byte(b Ball) []byte {
	jsonByte, _ := json.Marshal(b)
	return jsonByte
}

func Ball2String(b Ball) string {
	jsonByte, _ := json.Marshal(b)
	return string(jsonByte)
}

func String2Ball(ball string) Ball {
	var b Ball
	json.Unmarshal([]byte(ball), &b)
	return b
}

func Byte2Ball(unit []byte) Ball {
	var b Ball
	json.Unmarshal(unit, &b)
	return b
}
