package babyboy

import (
	"github.com/babyboy/babyboy/accounts"
	"github.com/babyboy/babyboy/node"
	"github.com/babyboy/babyboy/urfave/cli"
	"log"
)

type BabyEngine struct {
}

func NewBabyEngine() *BabyEngine {
	return &BabyEngine{}
}

func (engine *BabyEngine) InitEngine(ctx *cli.Context) error {

	// 初始化RpcServer
	fullNode := MakeFullNode(ctx)
	StartNode(fullNode)

	return nil
}

// startNode boots up the system node and all registered protocols, after which
// it unlocks any requested accounts, and starts the RPC/IPC interfaces and the
// miner.
func StartNode(stack *node.Node) error {

	// Start up the node itself
	if err := stack.Start(); err != nil {
		log.Printf("Error starting node: %v", err)
		return err
	}

	// Register wallet event handlers to open and auto-derive wallets
	events := make(chan accounts.WalletEvent, 16)
	stack.GetAccountManager().Subscribe(events)

	go func() {
		// Open any wallets already attached
		for _, wallet := range stack.GetAccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Println("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}
		// Listen for wallet event till termination
		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Println("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Println("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				//derivationPath := accounts.DefaultBaseDerivationPath
				//if event.Wallet.URL().Scheme == "ledger" {
				//	derivationPath = accounts.DefaultLedgerBaseDerivationPath
				//}
				//event.Wallet.SelfDerive(derivationPath, stateReader)

			case accounts.WalletDropped:
				log.Println("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()
	return nil
}
