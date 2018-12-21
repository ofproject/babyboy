package types

import (
	"github.com/babyboy/common"
)

type Inputs []Input

type Input struct {
	UnitHash     common.Hash `json:"unit"`
	MessageIndex int         `json:"message_index"`
	OutputIndex  int         `json:"ouput_index"`
	Type         string      `json:"type"`
	Output       Output      `json:"output"`
}

// NewInput
func NewInput(unitHash common.Hash, messageIndex int, outputIndex int, inType string, output Output) Input {
	return Input{
		UnitHash:     unitHash,
		MessageIndex: messageIndex,
		OutputIndex:  outputIndex,
		Type:         inType,
		Output:       output,
	}
}
