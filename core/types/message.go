package types

import (
	"github.com/babyboy/common"
	"fmt"
)

type Messages []Message

type Message struct {
	App         string      `json:"app"`
	PayloadHash common.Hash `json:"payloadhash"`
	Payload     Payload     `json:"payload"`
}

func (message Message) ToString() string {
	return fmt.Sprintf("\nmessage { \n App: %s \n Payloadhash: %s \n Payload: %s \n",
		message.App,
		message.PayloadHash.String(),
		message.Payload.ToString(),
	)
}

func NewMessage(app string, payloadHash common.Hash, payload Payload) Message {
	return Message{
		App:         app,
		PayloadHash: payloadHash,
		Payload:     payload,
	}
}

type MessageBuilder struct {
	App         string
	Payloadhash common.Hash
	Payload     Payload
}

func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{}
}

func (builder MessageBuilder) SetAppName(name string) *MessageBuilder {
	builder.App = name
	return &builder
}

func (builder MessageBuilder) GetAppName() string {
	return builder.App
}

func (builder MessageBuilder) SetPayloadHash(hash common.Hash) *MessageBuilder {
	builder.Payloadhash = hash
	return &builder
}

func (builder MessageBuilder) SetPayload(payload Payload) *MessageBuilder {
	builder.Payload = payload
	return &builder
}

func (builder MessageBuilder) GetMessage() Message {
	// TODO(ZXS) 做默认值的初始化工作, 类型的检查工作
	return NewMessage(builder.App, builder.Payloadhash, builder.Payload)
}
