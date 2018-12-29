package transaction

import "errors"

var (
	ErrNotUnSpentInput     = errors.New("未找到该笔交易的输入来源,请同步数据")
	ErrNotEnoughBalance    = errors.New("balance is not enough to spend for trading")
	ErrNotEnoughCommission = errors.New("balance is not enough to spend for commission")
	ErrNotFindAccount      = errors.New("unknown account")
	ErrUnitInfo            = errors.New("missing main information in the unit")
	ErrUnitInputs          = errors.New("valid unit inputs < 0")
	ErrUnitInputsLen       = errors.New("valid unit inputs len < 0")
	ErrUnitOutputs         = errors.New("valid unit outputs < 0")
	ErrUnitOutputsLen      = errors.New("valid unit outputs len < 0")
	ErrUnitAmountNoEqual   = errors.New("单元的输入和输出不对等")
	ErrUnitAuthorAddress   = errors.New("单元的输入未使用用户地址")
	ErrMainUpdate          = errors.New("MainChain update error")
	ErrPendingHandle       = errors.New("pending handle error")
	ErrUnitInfoValid       = errors.New("单元验证失败")
	ErrNotFindFrom         = errors.New("未找到输入来源的单元数据")
	ErrParentsList         = errors.New("单元的父节点不存在")
	ErrCheckUnitHash       = errors.New("单元的Hash校验错误")
	ErrTimeStamp           = errors.New("单元的时间戳小于父单元时间戳")
)
