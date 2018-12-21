package types

type ReqContent struct {
	Tag     string
	Command string
	Params  []string
}

type P2pRequest struct {
	ReqType string
	Content ReqContent
}

type BroadUnitEntity struct {
	HasPeers []string
	Message  string
}

type LightNewUnitEntity struct {
	FromAddress string
	ToAddress   string
	Amount      int
}

type LightNewUnitRepEntity struct {
	Error string
	Unit  Unit
}

type SyncDataEntity struct {
	State         int
	StableUnits   Units
	UnStableUnits Units
}

type PeerInfoEntity struct {
	CurrentPeerMCI int64
	UnStableUnits  int64
	MaxLevel       int64
}

type NewUnitEntity struct {
	FromPeerId string
	HasPeerIds []string
	NewUnit    Unit
}

type ValidMciEntity struct {
	MCI      int64
	UnitHash string
}
