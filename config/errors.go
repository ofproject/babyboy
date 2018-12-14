package config

import "errors"

var (
	ErrInitDbFailed     = errors.New("init database error")
	ErrInitP2PFailed    = errors.New("init P2P Server error")
	ErrInitRpcSerFailed = errors.New("init RpcServer error")
)
