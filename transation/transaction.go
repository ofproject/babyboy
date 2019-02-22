package transaction

import (
	"babyboy-dag/boydb"
	"github.com/babyboy/accounts"
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/config"
	"github.com/babyboy/core"
	"github.com/babyboy/core/types"
	"github.com/babyboy/dag"
	"github.com/babyboy/dag/memdb"
	"github.com/babyboy/event"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"sync"
)

const ConstHeaderCommission = 100
const ConstPayloadCommission = 100

type TXEventType int

const (
	NewUnitHandleDone TXEventType = iota
)

type TXEvent struct {
	NewUnitEntity types.NewUnitEntity // Wallet instance arrived or departed
	Kind          TXEventType         // Event type that happened in the system
}

func (tr *Transaction) Subscribe(sink chan<- TXEvent) event.Subscription {
	return tr.feed.Subscribe(sink)
}

type Transaction struct {
	ID          []byte
	Unit        types.Unit
	PendingProc *PendingPool
	StableProc  *StableProcess
	db          *boydb.DatabaseManager
	wg          sync.WaitGroup
	mux         sync.Mutex
	muxUnit     sync.Mutex
	chSubmitTx  chan types.NewUnitEntity
	feed        event.Feed
}

func NewTransaction() *Transaction {
	transaction := Transaction{}
	transaction.PendingProc = NewPendingPool()
	transaction.StableProc = NewStableProcess()

	transaction.db = boydb.GetDbInstance()

	transaction.chSubmitTx = make(chan types.NewUnitEntity, 16)
	go transaction.SubmitTXLoop(transaction.chSubmitTx)

	return &transaction
}

func (tr *Transaction) ChoiceInputs(from accounts.Account) []types.UTXO {
	var unSpent []types.UTXO
	var hasUseStableUnspent []types.UTXO

	pendingUnSpent := tr.FindUnspentTransactionFromPendingPool(from.Address)
	for _, value := range pendingUnSpent {
		unSpent = append(unSpent, value)

		lockUTXOs := tr.BackTrackingHasSpentUTXO(value)
		hasUseStableUnspent = append(hasUseStableUnspent, lockUTXOs...)
	}

	stableUnSpent := tr.FindUnspentTransactionFromStable(from.Address)
	for _, value := range stableUnSpent {
		if tr.isContantUTXO(value, hasUseStableUnspent) {
			continue
		} else {
			unSpent = append(unSpent, value)
		}
	}

	log.Println("被锁定的UTXO: ", len(hasUseStableUnspent))
	for _, u := range hasUseStableUnspent {
		strByte, _ := json.Marshal(u)
		log.Println(string(strByte))
	}
	log.Println("")

	return unSpent
}

func (tr *Transaction) isContantUTXO(target types.UTXO, all []types.UTXO) bool {
	result := false
	for _, value := range all {
		if target.ToHash() == value.ToHash() {
			result = true
		}
	}

	return result
}

func (tr *Transaction) buildTransactionUnit() types.Unit {

	db := boydb.GetDbInstance()
	pdb := memdb.GetParentMemDBInstance()
	wdb := memdb.GetWitnessMemDBInstance()

	gig := dag.NewGraphInfoGetter(db, pdb.GetParentsAsHash(), wdb.GetWitnessesAsHash())

	unit := types.NewEmptyUnit()
	unit.ParentList = pdb.GetParentsAsHash()
	unit.WitnessList = wdb.GetWitnessesAsHash()
	unit.BestParentUnit = gig.GetBestParentUnit()
	unit.Level = gig.GetLevel()
	unit.WitnessedLevel = gig.GetWitnessLevel()
	unit.LastBallUnit = gig.GetLastStableBall()
	unit.Authors = types.Authors{}
	unit.IsStable = false
	unit.MainChainIndex = 0
	unit.IsOnMainChain = false
	unit.SubStableMinHash = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	return unit
}

func (tr *Transaction) CreateTx(from accounts.Account, totalAmount *big.Int, tx string, amount int64) (types.Unit, error) {

	newUnit := tr.buildTransactionUnit()

	// Calculate all spending
	totalSpend := totalAmount.Int64() + ConstHeaderCommission + ConstPayloadCommission

	unSpent := tr.ChoiceInputs(from)

	var toOther []types.UTXO
	toMyself, curAmount := new(big.Int).SetInt64(0), new(big.Int).SetInt64(0)
	for _, u := range unSpent {
		curAmount = curAmount.Add(curAmount, u.Amount)
		toOther = append(toOther, u)

		if curAmount.Int64() >= totalSpend {
			toMyself = new(big.Int).Sub(curAmount, totalAmount)
			toMyself = new(big.Int).Sub(toMyself, new(big.Int).SetInt64(ConstHeaderCommission))
			toMyself = new(big.Int).Sub(toMyself, new(big.Int).SetInt64(ConstPayloadCommission))
			break
		}
	}

	if curAmount.Int64() < totalAmount.Int64() {
		return types.Unit{}, ErrNotEnoughBalance
	}

	if curAmount.Int64() < totalSpend {
		return types.Unit{}, ErrNotEnoughCommission
	}

	var inputs types.Inputs
	for _, u := range toOther {
		input := types.NewInput(u.UnitHash, u.MessageIndex, u.OutputIndex, u.Type, u.Amount)
		//strByte, _ := json.Marshal(input)
		//log.Println(string(strByte))
		inputs = append(inputs, input)
	}

	var outputs types.Outputs
	outputs = append(outputs, types.NewOutput(common.HexToAddress(tx), big.NewInt(amount)))

	if toMyself.Cmp(new(big.Int).SetInt64(0)) > 0 {
		outputs = append(outputs, types.NewOutput(from.Address, toMyself))
	}

	payload := types.NewPayLoad().
		AddInputs(inputs).
		AddOutputs(outputs)

	payloadHash := payload.GetPayloadHash()

	builder := types.NewMessageBuilder().
		SetAppName(config.Const_Message_AppType_Payment).
		SetPayloadHash(payloadHash).
		SetPayload(payload)

	messages := types.Messages{builder.GetMessage()}
	newUnit.Messages = messages

	newUnit.PayloadCommission = big.NewInt(ConstPayloadCommission)
	newUnit.HeadersCommission = big.NewInt(ConstHeaderCommission)

	return newUnit, nil
}

func (tr *Transaction) PendingTx(unit types.Unit) error {
	err := tr.PendingProc.HandleUnit(unit)
	return err
}

func (tr *Transaction) StableTx(unit types.Unit) ([]types.Commission, bool, error) {
	commissions, valid, err := tr.StableProc.HandleUnit(unit)
	return commissions, valid, err
}

func (tr *Transaction) RecvUnit(newUnitEntity types.NewUnitEntity) {
	tr.chSubmitTx <- newUnitEntity
}

func (tr *Transaction) SubmitTXLoop(chSubmitTx <-chan types.NewUnitEntity) {
	for newUnitEntity := range chSubmitTx {
		newUnit := newUnitEntity.NewUnit

		if signer := core.NewSigner(); !signer.VerifyUnit(newUnit) {
			log.Println("签名验证失败,不进行存储和广播")
			continue
		}

		if err := tr.ReviewUnit(newUnit); err != nil {
			log.Println(err)
			continue
		}

		if err := tr.HandlerNewUnit(newUnit); err != nil {
			log.Println(err)
		}

		tr.feed.Send(TXEvent{Kind: NewUnitHandleDone, NewUnitEntity: newUnitEntity})
	}
}

func (tr *Transaction) ReviewUnit(unit types.Unit) error {

	isExist := boydb.GetDbInstance().IsExistUnit(unit.Hash)
	if isExist {
		return errors.New("已经存在该单元")
	}
	if err := tr.ValidUTXOAmount(unit); err != nil {
		return err
	} else {
		log.Println("单元内输入输出金额正确")
	}
	tr.muxUnit.Lock()

	mc := NewMainChain()
	if err := mc.ValidUnit(unit); err != nil {
		log.Println("单元验证失败,跟最优节点核对主链信息")
		//eventbus.GetEventBus().Publish("node:ValidUnitErr", )
		tr.muxUnit.Unlock()
		return err
	}
	tr.muxUnit.Unlock()
	tr.BackTrackingUTXO(unit)

	return nil
}
