package baby

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/log"
	"github.com/babyboy/node"
	"github.com/babyboy/p2p"
	"github.com/babyboy/p2p/nat"
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

func (n *Node) Start() error {
	babyboy.StartNode(n.node)
	return nil
}

func (n *Node) NewAccount(password string) string {
	address, err := n.node.NewAccount(password)
	if err != nil {
		log.Error("NewAccount Error", err)
	}

	return address.String()
}

func (n *Node) NewJoint(address string, password string, tx string, amount int64) string {
	hash, err := n.node.NewJoint(address, password, tx, amount)
	if err != nil {
		log.Error("SendTx Error: ", err)
	}

	return hash.String()
}

func (n *Node) GetMaxLevel() int64 {
	level := n.node.ProtocolManager.GetMaxLevel()

	return level
}

func (n *Node) ListAccounts() string {
	address := n.node.ListAccounts()
	if len(address) == 0 {
		return ""
	}

	return address[0].String()
}

func (n *Node) Wallets() int {
	count := n.node.GetWallets()

	return count
}

type WalletBalance struct {
	address string
	amount  int64
}

func (n *Node) GetBalanceS(address string) int64 {
	b, err := n.node.GetBalance(address)
	if err != nil {
		log.Error("GetBalance Error", err)
	}

	return b.Stable
}

type FilterLogsHandler interface {
	OnError(failure string)
}

func (n *Node) CallbackFunc(cb FilterLogsHandler) int64 {
	cb.OnError("Hello")

	return 0
}
