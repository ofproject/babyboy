package config

import (
	"babyboy/common"
	"babyboy/core/types"
)

// Creation unit related constant
const GENISIS_UNIT_HASH = "0x1ba940d64a71b77bf3ffee246ed2d025c00034e0b5f3719b48825d2106afbe1a"

var WitnessList = []string{
	"0x2f20c4d83b2c69154255228695bb4ab5ebaaa30b",
	"0x1017035ee3ab7aaa2c05deea977c143eac69c8a5",
	"0x3415e9a9e6d42efd87cb96fc2527af1379de53f8",
	"0x760d32f612f3bb2923adf384e8a62db82a073c36",
	"0xd1b4f9aa6c4a35872c6747bd985b5feebdc24cc5",
	"0xc3e9e7c5bf437b67a5ef00207e2edd8107de6e89",
	"0x5f4af14b320137d2cd60369d7da52ce1e22337cd",
	"0xf9f202b2d660db24b692c38050b0da35f54980a2",
	"0xe59c63b55781542af0983664e57e31818429c802",
	"0x22cabd61e1a590b558bac3977bacabc6d9fae65b",
	"0x402e27195a11f38476b188c4615c616f87e0943e",
	"0x82606c6d81cc768b7e3e3a31f9c1c7cb062d4529",
}

// Create Genesis Unit
func GenesisUnit() types.Unit {
	var u types.Unit
	u.Version = "1.0"
	u.Messages = nil
	u.Authors = make(types.Authors, 1)

	u.ParentList = nil
	u.LastBallUnit = common.Hash{}
	u.Authors[0].Address = common.HexToAddress(WitnessList[0])
	u.WitnessList = []common.Address{
		common.HexToAddress(WitnessList[0]),
		common.HexToAddress(WitnessList[1]),
		common.HexToAddress(WitnessList[2]),
		common.HexToAddress(WitnessList[3]),
		common.HexToAddress(WitnessList[4]),
		common.HexToAddress(WitnessList[5]),
		common.HexToAddress(WitnessList[6]),
		common.HexToAddress(WitnessList[7]),
		common.HexToAddress(WitnessList[8]),
		common.HexToAddress(WitnessList[9]),
		common.HexToAddress(WitnessList[10]),
		common.HexToAddress(WitnessList[11]),
	}
	u.Level = 0
	u.WitnessedLevel = 0
	u.HeadersCommission = 0
	u.PayloadCommission = 0
	u.IsStable = true
	u.MainChainIndex = 0
	u.IsOnMainChain = true
	u.BestParentUnit = common.Hash{}
	var outputs types.Outputs
	for i := 0; i < len(WitnessList); i++ {
		output := types.NewOutput(u.WitnessList[i], 10000000)
		// Build two output transactions
		outputs = append(outputs, output)
	}

	// Construct a message payload
	payload := types.NewPayLoad().
		AddOutput(outputs[0]).
		AddOutput(outputs[1]).
		AddOutput(outputs[2]).
		AddOutput(outputs[3]).
		AddOutput(outputs[4]).
		AddOutput(outputs[5]).
		AddOutput(outputs[6]).
		AddOutput(outputs[7]).
		AddOutput(outputs[8]).
		AddOutput(outputs[9]).
		AddOutput(outputs[10]).
		AddOutput(outputs[11])

	payloadHash := payload.GetPayloadHash()

	// Build a message entity
	builder := types.NewMessageBuilder().
		SetAppName(Const_Message_AppType_Payment).
		SetPayloadHash(payloadHash).
		SetPayload(payload)

	messages := types.Messages{builder.GetMessage()}
	u.Messages = messages
	u.SubStableMinHash = common.HexToHash(GENISIS_UNIT_HASH)
	u.Hash = common.HexToHash(GENISIS_UNIT_HASH)
	return u
}
