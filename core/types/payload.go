package types

import (
	"encoding/json"
	"fmt"

	"github.com/babyboy/common"
	"github.com/babyboy/crypto/sha3"
)

type Payload struct {
	Inputs  Inputs  `json:"inputs"`
	Outputs Outputs `json:"outputs"`
}

func (pay Payload) ToString() string {
	return fmt.Sprintf("{ \n Inputs: %s \n Outputs: %s}\n",
		pay.Inputs, pay.Outputs)
}

func NewPayLoad() *Payload {
	return &Payload{}
}

func (load Payload) AddInput(input ...Input) Payload {
	for _, in := range input {
		load.Inputs = append(load.Inputs, in)
	}
	return load
}

func (load Payload) AddInputs(inputs Inputs) Payload {
	for _, in := range inputs {
		load.Inputs = append(load.Inputs, in)
	}
	return load
}

func (load Payload) AddOutputs(outputs Outputs) Payload {
	for _, out := range outputs {
		load.Outputs = append(load.Outputs, out)
	}
	return load
}

func (load Payload) AddOutput(output ...Output) Payload {
	for _, out := range output {
		load.Outputs = append(load.Outputs, out)
	}
	return load
}

func (load Payload) GetInputs() Inputs {
	return load.Inputs
}

func (load Payload) GetOutPuts() Outputs {
	return load.Outputs
}

func (load Payload) GetPayloadHash() common.Hash {
	jsonByte, _ := json.Marshal(load)
	hash := sha3.Sum256(jsonByte)
	return hash
}
