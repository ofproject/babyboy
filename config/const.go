package config

import (
	"math"
	"time"
)

// witness
var CountWitness = 12
var MajorityOfWitnesses int

func init() {
	if CountWitness%2 == 0 {
		MajorityOfWitnesses = (CountWitness / 2) + 1
	} else {
		MajorityOfWitnesses = int(math.Ceil(float64(CountWitness) / 2.0))
	}
}

const Const_Stable_Rounds = 1

// NetWork About
const NETWORK_ID = 1
const HAND_SHAKE_TIMEOUT = 5 * time.Second

const Const_DATABASE_NAME = "dagdata"
const Const_DATABASE_PATH = "leveldb/"

// Database storage related constants
const ConstDBParentListPrefix = "pl."
const ConstDBWitnessListPrefix = "wl."
const ConstDBUnitPrefix = "u."
const ConstDBBallPrefix = "b."
const ConstDBOutputPrefix = "o."
const ConstDBPendingUnitPrefix = "pu."
const ConstDBChildrenHash = "children."
const ConstDBStableUnitsPrefix = "su."
const ConstDBVoteRound = "vote_round."
const ConstDBVoteResult = "vote_result."
const ConstCacheUnit = "cache."

// Transaction related constant
const Const_Message_AppType_Payment = "payment"
const ConstMaxHash = "0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"
