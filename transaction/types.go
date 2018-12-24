package transaction

import (
	"github.com/babyboy/core/types"
	"github.com/babyboy/common"
)

type ResultBack struct {
	stable bool
	exist  bool
	utxo   types.UTXO
}

type UtxoHelper struct {
	Address  common.Address
	UTXO     types.UTXO
	IsStable bool
}
