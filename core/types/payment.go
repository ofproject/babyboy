package types

import (
	"github.com/babyboy/common"
)

type PaymentBuilder struct {
	payment Payload
	message Messages
	inputs  Inputs
	outputs Outputs
}

func NewParentUnits(hash ...common.Hash) []common.Hash {
	parentList := make([]common.Hash, 0)
	for _, h := range hash {
		parentList = append(parentList, h)
	}
	return parentList
}
