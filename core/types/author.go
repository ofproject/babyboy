package types

import (
	"github.com/babyboy/common"
	"fmt"
)

type Authors []Author

type Author struct {
	Address   common.Address `json:"address"`
	Signature []byte         `json:"signature"`
	//PublicKey []byte		 `json:"publickey"`
}

func (au Author) ToString() string {
	return fmt.Sprintf("{ \n Address: %s \n Signature: %s}\n",
		au.Address, au.Signature)
}

func NewAuthor(address common.Address, signature []byte) Author {
	return Author{
		Address:   address,
		Signature: signature,
		//PublicKey: publicKey,
	}
}
