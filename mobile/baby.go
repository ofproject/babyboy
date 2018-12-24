package baby

import (
	"github.com/babyboy/babyboy"
	"github.com/babyboy/node"
	"github.com/babyboy/p2p"
	"github.com/babyboy/log"
	"github.com/babyboy/p2p/nat"
	"github.com/babyboy/common"
	"encoding/json"
	"github.com/babyboy/config"
)

type Node struct {
	node *node.Node
}

func NewNode(dataDir string) (stack *Node, _ error) {

	cfg := babyboy.BabyConfig{
		Node: node.Config{
			DataDir:   dataDir,
			RpcServer: "0.0.0.0:8545",
			P2P: p2p.Config{
				ListenAddr: ":3000",
				MaxPeers:   25,
				NAT:        nat.Any(),
			},
		},
	}

	fullNode, err := node.New(&cfg.Node)
	if err != nil {
		return &Node{}, err
	}

	return &Node{fullNode}, nil
}

func (n *Node) SyncData() {
	go n.node.GetProtocolMgr().From0SyncData()
	return
}

type FilterLogsHandler interface {
	OnError(failure string)
}

func (n *Node) CallbackFunc(cb FilterLogsHandler) int64 {
	cb.OnError("Hello")

	return 0
}

func (n *Node) Ip() string {
	ser := n.node.Server()
	if ser == nil {
		return ""
	}

	return ser.NodeInfo().IP
}

type WalletBalance struct {
	address string
	amount  int64
}

func (n *Node) Start() error {
	if err := babyboy.StartNode(n.node); err != nil {
		return err
	}

	return nil
}

func (n *Node) NewAccount(password string) string {
	address, err := n.node.NewAccount(password)
	if err != nil {
		log.Error("NewAccount Error", err)
	}

	return address.String()
}

func (n *Node) NewJoint(from string, to string, password string, amount int) string {
	hash, err := n.node.NewJoint(from, password, to, amount)
	if err != nil {
		log.Error("SendTx Error: ", err)
	}

	return hash.String()
}

func (n *Node) GetBalanceStable(address string) int64 {
	b, err := n.node.GetBalance(address)
	if err != nil {
		log.Error("GetBalance Error", err)
	}

	return b.Stable
}

func (n *Node) GetMaxLevel() int64 {
	db := n.node.GetProtocolMgr()
	if db == nil {
		return -1
	}

	return db.GetMaxLevel()
}

func (n *Node) ListAccounts() string {
	address := n.node.ListAccounts()
	if len(address) == 0 {
		return "address is null"
	}

	return address[0].String()
}

func (n *Node) Wallets() int {
	count := n.node.GetWallets()

	return count
}

func (n *Node) PeerCount() int64 {
	ser := n.node.Server()
	if ser == nil {
		return int64(-1)
	}

	return int64(ser.PeerCount())
}

func (n *Node) Peers() string {
	ser := n.node.Server()
	if ser == nil {
		return "ser is null"
	}

	type PeerInfo struct {
		ID            string `json:"id"`
		RemoteAddress string `json:"remote_address"`
	}

	type peersStruct struct {
		Peers []PeerInfo `json:"peers"`
	}

	var peersInfos []PeerInfo
	for _, value := range ser.PeersInfo() {
		var p PeerInfo
		p.ID = value.ID
		p.RemoteAddress = value.Network.RemoteAddress

		peersInfos = append(peersInfos, p)
	}

	var peers = peersStruct{
		Peers: peersInfos,
	}

	jsonPeers, err := json.Marshal(peers)
	if err != nil {
		return "json is err"
	}

	return string(jsonPeers)
}

func (n *Node) NodeInfo() string {
	ser := n.node.Server()
	if ser == nil {
		return "ser is null"
	}

	return ser.NodeInfo().Enode
}

func (n *Node) GetWitnessList() string {
	witness := n.node.GetWitness()
	if witness == nil {
		return "witness is null"
	}

	type witnessStruct struct {
		Witness []string `json:"witness"`
	}

	var w = witnessStruct{
		Witness: witness,
	}

	jsonWitness, err := json.Marshal(w)
	if err != nil {
		return "json is err "
	}

	return string(jsonWitness)
}

func (n *Node) GetUnitInfoByHash(hash string) string {
	db := n.node.GetDbMgr()
	if db == nil {
		return "db is null"
	}

	uHash := common.HexToHash(hash)
	unit, err := db.GetUnitByHash(uHash)
	if err != nil {
		return "unit is null"
	}

	type unitStuct struct {
		Hash           string `json:"hash"`
		ParentUnit     string `json:"parent_unit"`
		Address        string `json:"create_address"`
		BestParentUnit string `json:"best_parent_unit"`
		IsStable       bool   `json:"is_stable"`
		Level          int64  `json:"level"`
		WitnessedLevel int64  `json:"witnessed_level"`
	}

	var uStruct unitStuct
	uStruct.WitnessedLevel = unit.WitnessedLevel
	uStruct.BestParentUnit = unit.BestParentUnit.String()
	uStruct.Level = unit.Level
	uStruct.Hash = unit.Hash.String()
	if hash != config.GENISIS_UNIT_HASH {
		uStruct.ParentUnit = unit.ParentList[0].String()
		uStruct.Address = unit.Authors[0].Address.String()
	}
	uStruct.IsStable = unit.IsStable

	jsonUnit, err := json.Marshal(uStruct)
	if err != nil {
		return "json is err"
	}

	return string(jsonUnit)
}

func (n *Node) GetTransactionInfoByHash(hash string) string {
	db := n.node.GetDbMgr()
	if db == nil {
		return "db is null"
	}

	uHash := common.HexToHash(hash)
	unit, err := db.GetUnitByHash(uHash)
	if err != nil {
		return "unit is null"
	}

	type OutputStruct struct {
		Address string `json:"address"`
		Amount  int    `json:"amount"`
	}

	type InsertStruct struct {
		From   string `json:"from"`
		Amount int    `json:"amount"`
	}

	type unitStuct struct {
		Payloadhash string         `json:"transaction_hash"`
		Instert     []InsertStruct `json:"input"`
		OutPut      []OutputStruct `json:"output"`
	}

	var uStruct unitStuct
	var uInsert []InsertStruct
	var uOutput []OutputStruct

	uStruct.Payloadhash = unit.Messages[0].PayloadHash.String()

	for _, value := range unit.Messages[0].Payload.Inputs{
		var insertUnit InsertStruct
		insertUnit.From = value.Output.Address.String()
		insertUnit.Amount = value.Output.Amount

		uInsert = append(uInsert, insertUnit)
	}

	for _, value := range unit.Messages[0].Payload.Outputs {
		var uOutPut OutputStruct
		uOutPut.Address = value.Address.String()
		uOutPut.Amount = value.Amount

		uOutput = append(uOutput, uOutPut)
	}

	uStruct.Instert = uInsert
	uStruct.OutPut = uOutput

	jsonTX, err := json.Marshal(uStruct)
	if err != nil {
		return "json is err"
	}

	return string(jsonTX)
}

func (n *Node) GetStableCount() int64 {
	dbMger := n.node.GetDbMgr()
	if dbMger == nil {
		return -1
	}

	return dbMger.GetAllStableUnitCount()
}
