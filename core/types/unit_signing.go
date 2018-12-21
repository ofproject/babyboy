package types

import (
	"github.com/babyboy/common"
)

type Signer interface {
	Hash(unit *Unit) common.Hash
	PublicKey(unit *Unit) ([]byte, error)
	Sign(unit *Unit) ([]byte, error)
}

type BabySigner struct{}

func (b BabySigner) Hash(unit *Unit) common.Hash {
	return RlpHash([]interface{}{
		unit.Messages,
		unit.LastBallUnit,
		unit.ParentList,
		unit.WitnessList,
	})
}
