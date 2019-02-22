package transaction

import (
	"babyboy-dag/core/types"
	"babyboy-dag/common"
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
