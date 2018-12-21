package types

import "github.com/babyboy/common"

type HashArray struct {
	Hashes []common.Hash
}

func NewHashArray() HashArray {
	return HashArray{}
}
