package node

import (
	"errors"
)

var (
	ErrNodeSender    = errors.New("请填写发起人地址")
	ErrNodePassWord  = errors.New("请输入密码")
	ErrNodeAmount    = errors.New("请填写要发送到地址和金额")
	ErrNodeNoAccount = errors.New("FindAccount Error")
	ErrNodeCreateTX  = errors.New("CreateTX Error")
	ErrNodeSinged    = errors.New("SingedUnit Error")
	ErrAmountRange   = errors.New("AmountRange Error")
	ErrLockAccount   = errors.New("LockAccount Error")
)
