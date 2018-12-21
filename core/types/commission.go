package types

import (
	"github.com/babyboy/common"
)

type Commissions []Commission

type Commission struct {
	Address common.Address
	UTXO    UTXO
}

func NewCommission(address common.Address, utxo UTXO) Commission {
	return Commission{Address: address, UTXO: utxo}
}
