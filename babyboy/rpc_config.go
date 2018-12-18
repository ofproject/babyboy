package babyboy

import (
	"github.com/babyboy/babyboy/utils"
	"github.com/babyboy/babyboy/node"
	"github.com/babyboy/babyboy/urfave/cli"
	"log"
)

type BabyConfig struct {
	Node node.Config
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	return cfg
}

func MakeFullNode(ctx *cli.Context) *node.Node {
	stack := makeConfigNode(ctx)
	utils.RegisterBabyService(stack)

	return stack
}

func makeConfigNode(ctx *cli.Context) *node.Node {

	// Load defaults.
	cfg := BabyConfig{
		Node: defaultNodeConfig(),
	}

	if ctx != nil && ctx.GlobalIsSet(utils.RpcPortFlag.Name) {
		rpcPort := ctx.GlobalString(utils.RpcPortFlag.Name)
		cfg.Node.RpcServer = "0.0.0.0:" + rpcPort
	}

	if ctx != nil && ctx.GlobalIsSet(utils.DataDirFlag.Name) {
		cfg.Node.DataDir = ctx.GlobalString(utils.DataDirFlag.Name)
	}

	if ctx != nil && ctx.GlobalIsSet(utils.P2pPortFlag.Name) {
		addr := ":" + ctx.GlobalString(utils.P2pPortFlag.Name)
		cfg.Node.P2P.ListenAddr = addr
	}

	if ctx != nil && ctx.GlobalIsSet(utils.NoDiscoverFlag.Name) {
		cfg.Node.P2P.NoDiscovery = true
	}

	stack, err := node.New(&cfg.Node)
	if err != nil {
		log.Println("Failed to create the protocol stack: ", err)
		return &node.Node{}
	}

	return stack
}
