package types

import (
	"github.com/babyboy/common"
	"fmt"
)

type Outputs []Output

// Special to payment Message, which included the receiver and money
type Output struct {
	Address common.Address `json:"address"`
	Amount  int            `json:"amount"`
}

func (out Output) ToString() string {
	return fmt.Sprintf("\nonput { \n Address: %s \n Amount : %d \n}\n",
		out.Address.String(), out.Amount)
}

func NewOutput(address common.Address, amount int) Output {
	return Output{
		Address: address,
		Amount:  amount,
	}
}
