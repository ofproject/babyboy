package transaction

import "errors"

var (
	ErrNotUnSpentInput     = errors.New("The source of input for this transaction was not found. Please synchronize the data.")
	ErrNotEnoughBalance    = errors.New("balance is not enough to spend for trading")
	ErrNotEnoughCommission = errors.New("balance is not enough to spend for commission")
	ErrNotFindAccount      = errors.New("unknown account")
	ErrUnitAmountNoEqual   = errors.New("Input and output of cells are not equal")
)
