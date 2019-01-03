package node

import (
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"sync"

	"babyboy-dag/accounts"
	"babyboy-dag/accounts/keystore"
	"babyboy-dag/boy"
	"babyboy-dag/boydb"
	"babyboy-dag/common"
	"babyboy-dag/common/hexutil"
	"babyboy-dag/common/queue"
	"babyboy-dag/config"
	"babyboy-dag/core/types"
	"babyboy-dag/crypto"
	"babyboy-dag/dag"
	"babyboy-dag/dag/memdb"
	"babyboy-dag/event"
	"babyboy-dag/eventbus"
	"babyboy-dag/p2p"
	"babyboy-dag/p2p/discover"
	"babyboy-dag/rpc"
	"babyboy-dag/transaction"
	"encoding/json"
	"path"
	"sort"
	"strconv"
	"strings"
)

type State int

const (
	Running State = iota
	Stopped
	SynchronizingRecving
	SynchronizingReponse
	PushUnits
)

// Node is a container on which services can be registered.
type Node struct {
	eventmux          *event.TypeMux // Event multiplexer used between the services of a stack
	accmgr            *accounts.Manager
	config            *Config
	ephemeralKeystore string                   // if non-empty, the key directory that will be removed by Stop
	serviceFuncs      []ServiceConstructor     // Service constructors (in dependency order)
	services          map[reflect.Type]Service // Currently running services
	rpcAPIs           []rpc.API                // List of APIs currently provided by the node
	lock              sync.RWMutex
	transaction       *transaction.Transaction
	protocolManager   *boy.ProtocolManager
	dbManager         *boydb.DatabaseManager
	server            *p2p.Server // Currently running P2P networking layer
	state             State
	recvQueue         *queue.Queue // 同步时接收数据队列
	waitQueue         *queue.Queue // 同步时收到其他p2p广播的数据时缓存队列
	syncCount         int
	chain             map[common.Hash]*types.DagBlock
}

// New creates a new P2P node, ready for protocol registration.
func New(conf *Config) (*Node, error) {
	// Copy config and resolve the datadir so future changes to the current
	// working directory don't affect the node.
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		absdatadir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = absdatadir
	}

	// Ensure that the AccountManager method works before the node has started.
	// We rely on this in cmd/geth.
	am, ephemeralKeystore, err := makeAccountManager(conf)
	if err != nil {
		return nil, err
	}
	// Note: any interaction with Config that would create/touch files
	// in the data directory or instance directory is delayed until Start.
	return &Node{
		accmgr:            am,
		ephemeralKeystore: ephemeralKeystore,
		config:            conf,
		serviceFuncs:      []ServiceConstructor{},
		transaction:       transaction.NewTransaction(),
		recvQueue:         queue.New(),
		waitQueue:         queue.New(),
	}, nil
}

// Register injects a new service into the node's stack. The service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.serviceFuncs = append(n.serviceFuncs, constructor)

	return nil
}

// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.initDatabase()

	protocol, err := n.initP2p()
	if err != nil {
		return err
	}
	n.protocolManager = protocol

	services, err := n.initServices()
	if err != nil {
		return err
	}

	if err := n.startRPC(services); err != nil {
		return err
	}

	n.initEventBus()
	n.initGenesis()

	//G, _ := n.dbManager.GetUnitByHash(common.HexToHash(config.GENISIS_UNIT_HASH))
	//n.InitDag(G)

	return nil
}

func (n *Node) initServices() (map[reflect.Type]Service, error) {

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular service
		ctx := &ServiceContext{
			Services:       make(map[reflect.Type]Service),
			EventMux:       n.eventmux,
			AccountManager: n.accmgr,
			Node:           n,
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.Services[kind] = s
		}
		// Construct and save the service
		service, err := constructor(ctx)
		if err != nil {
			return nil, err
		}
		kind := reflect.TypeOf(service)

		services[kind] = service
	}

	return services, nil
}

func (n *Node) initEventBus() {

	// Register P2p event handlers to P2p Event
	p2pevents := make(chan boy.P2pEvent, 16)
	n.protocolManager.Subscribe(p2pevents)
	go func() {
		for p2pevent := range p2pevents {
			switch p2pevent.Kind {
			case boy.NewUnitReceived:
				entity := p2pevent.Data.(types.NewUnitEntity)
				n.handleNewUnitEvent(entity)
			case boy.NewNodeConnect:
				//log.Println("新节点连接, 将Cache发送过去")
				p := p2pevent.Data.(*boy.Peer)
				// TODO 暂时先强制只要有连接就同步一次不稳定点
				pdb := memdb.GetParentMemDBInstance()
				wdb := memdb.GetWitnessMemDBInstance()
				graphInfo := dag.NewGraphInfoGetter(n.dbManager, pdb.GetDagAllTips(), wdb.GetWitnessesAsHash())
				mci := graphInfo.GetLastStableBallMCI()
				units, uUnits := graphInfo.GetMissingUnits(mci, mci)
				units = append(units, uUnits...)
				sort.Sort(units)
				for _, unit := range units {
					bUnit, _ := json.Marshal(unit)
					if unit.Hash.String() == config.GENISIS_UNIT_HASH {
						continue
					}
					entity := &types.BroadUnitEntity{HasPeers: n.protocolManager.GetPeers().GetPeersIds(), Message: string(bUnit)}
					n.protocolManager.SendMsgToPeer(p, boy.MSG_NewUnit, entity)
				}
			case boy.SyncDataRequst:
				entity := p2pevent.Data.(boy.SyncDataRequestEntity)
				go n.pickDataFromDag(entity.ReqPeer, entity.MCI)
			}
		}
	}()

	// Register Transaction event handlers to TX Event
	txevents := make(chan transaction.TXEvent, 16)
	n.transaction.Subscribe(txevents)
	go func() {
		for txevent := range txevents {
			switch txevent.Kind {
			case transaction.NewUnitHandleDone:
				n.handleUnitDoneEvent(txevent.NewUnitEntity)
			}
		}
	}()

	eventbus.GetEventBus().Subscribe("node:SyncUnit", func() {
		if n.state == Running {
			n.protocolManager.SetIsRequireSync(false)
			go n.protocolManager.Synchronise(n.protocolManager.GetBestPeer())
		}
	})

	eventbus.GetEventBus().Subscribe("node:SyncDataReq", func(p *boy.Peer, endMci string) {
		log.Println("EventBus: ", "node:SyncDataReq")
		if n.state == SynchronizingReponse {
			log.Println("Syncing...")
			n.notifyBusy(p)
			return
		}

		n.state = SynchronizingReponse

		go n.pickDataFromDag(p, endMci)
	})

	eventbus.GetEventBus().Subscribe("node:SyncDataRep", func(p *boy.Peer, syncData types.SyncDataEntity) {
		//log.Println("EventBus: ", "node:SyncDataRep")
		if syncData.State == 2 {
			log.Println("Request Busy...")
			n.state = Running
			n.protocolManager.SetIsRequireSync(true)
			return
		}

		n.state = SynchronizingRecving
		// 处理稳定单元
		for _, unit := range syncData.StableUnits {
			_, err := boydb.GetDbInstance().GetUnitByHash(unit.Hash)
			if err == nil {
				log.Println("Unit has exsit")
				continue
			}

			n.showProgress(int64(unit.Level), p.GetMaxLevel())

			entity := types.NewUnitEntity{FromPeerId: "", HasPeerIds: []string{}, NewUnit: unit}
			n.recvQueue.Push(entity)
		}

		if syncData.State == 1 {
			log.Println("Sync Data End")
			go n.fetchData()
		}
	})

	eventbus.GetEventBus().Subscribe("node:LightNewUnit", func(entity types.LightNewUnitEntity, callback func(unit types.Unit, err error)) {
		unit, err := n.CreateUnitForLight(entity.FromAddress, entity.ToAddress, entity.Amount)
		callback(unit, err)
	})
}

func (n *Node) handleNewUnitEvent(entity types.NewUnitEntity) {
	//log.Println("New Unit Message: ", entity.NewUnit.Level)

	if n.state == SynchronizingRecving {
		n.waitQueue.Push(entity)
		return
	}

	if n.protocolManager.GetPeers().Len() == 0 {
		//log.Println("当前未连接任何节点, 先将消息缓存在数据库中")
		n.dbManager.SaveCacheUnitToDb(entity.NewUnit)

		units := n.dbManager.GetCacheUnitFromDb()
		log.Println(len(units))
	}
	n.transaction.RecvUnit(entity)
}

func (n *Node) handleUnitDoneEvent(entity types.NewUnitEntity) {
	//log.Println("EventBus: ", "node:HandleUnitDone")

	//n.DagAddUnit(entity.NewUnit, entity.NewUnit.ParentList, n.chain)
	//tips := n.FindTips(n.chain)
	//log.Println("顶点个数: ", len(tips))
	//n.countBlue(n.chain, entity.NewUnit)

	unitByte, err := json.Marshal(entity.NewUnit)
	if err != nil {
		log.Println(err)
		return
	}

	joint := &types.BroadUnitEntity{HasPeers: n.protocolManager.GetPeers().GetPeersIds(), Message: string(unitByte)}
	jointByte, err := json.Marshal(joint)
	if err != nil {
		log.Println(err)
		return
	}
	if entity.FromPeerId == "local" {
		n.protocolManager.SendMsgToPeers(boy.MSG_NewUnit, string(jointByte))
	} else {
		n.protocolManager.BroadMsgExcludePeer(entity.HasPeerIds, boy.MSG_NewUnit, string(jointByte))
	}
}

func (n *Node) pickDataFromDag(p *boy.Peer, endmci string) {

	pdb := memdb.GetParentMemDBInstance()
	wdb := memdb.GetWitnessMemDBInstance()
	graphInfo := dag.NewGraphInfoGetter(n.dbManager, pdb.GetDagAllTips(), wdb.GetWitnessesAsHash())
	startMci := graphInfo.GetLastStableBallMCI()
	endMci, err := strconv.ParseInt(endmci, 10, 64)
	if err != nil {
		log.Println(err)
		return
	}
	sUnits, uUnits := graphInfo.GetMissingUnits(startMci, endMci)
	sUnits = append(sUnits, uUnits...)
	//log.Println("同步数据开始: 稳定点数量: ", len(sUnits), "  不稳定点数量: ", len(uUnits))
	for _, unit := range sUnits {
		msg := types.SyncDataEntity{State: 0, StableUnits: types.Units{unit}, UnStableUnits: types.Units{}}
		if err := n.protocolManager.SendMsgToPeer(p, boy.MSG_SYNC_REP, msg); err != nil {
			log.Println("传输异常中断")
			break
		}
	}

	// 通知接收节点同步结束
	msg := types.SyncDataEntity{State: 1, StableUnits: types.Units{}, UnStableUnits: types.Units{}}
	if err := n.protocolManager.SendMsgToPeer(p, boy.MSG_SYNC_REP, msg); err != nil {
		log.Println("同步数据结束")
	}

	n.state = Running
}

func (n *Node) fetchData() {
	for !n.recvQueue.Empty() {
		entity := n.recvQueue.Front().(types.NewUnitEntity)
		n.recvQueue.Pop()

		newUnit := entity.NewUnit

		bs, _ := json.Marshal(newUnit)
		log.Println(string(bs))
		// 验证单元是否合格
		if err := n.transaction.ReviewUnit(newUnit); err != nil {
			log.Println(err)
			continue
		}

		// 处理一笔新交易
		if err := n.transaction.HandlerNewUnit(newUnit); err != nil {
			log.Println(err)
		}
		//time.Sleep(time.Millisecond * 2000)
	}

	// 同步期间可能会有新的交易过来，先加入到等待队列中，同步完成后一起处理
	for !n.waitQueue.Empty() {
		entity := n.waitQueue.Front().(types.NewUnitEntity)
		n.waitQueue.Pop()

		newUnit := entity.NewUnit
		// 验证单元是否合格
		if err := n.transaction.ReviewUnit(newUnit); err != nil {
			log.Println(err)
			continue
		}

		// 处理一笔新交易
		if err := n.transaction.HandlerNewUnit(newUnit); err != nil {
			log.Println(err)
			continue
		}

		n.handleUnitDoneEvent(entity)
		//time.Sleep(time.Millisecond * 1000)
	}

	n.state = Running
	n.protocolManager.SetIsRequireSync(true)
}

func (n *Node) notifyBusy(p *boy.Peer) {
	// 通知接收节点同步结束
	msg := types.SyncDataEntity{State: 2, StableUnits: types.Units{}, UnStableUnits: types.Units{}}
	if err := n.protocolManager.SendMsgToPeer(p, boy.MSG_SYNC_REP, msg); err != nil {
		log.Println(err)
	}
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(apis []rpc.API) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	endpoint := n.config.RpcServer

	_, _, err := rpc.StartHTTPEndpoint(endpoint, apis, []string{}, []string{}, []string{})
	if err != nil {
		return err
	}

	log.Println("HttpSever Listening on", fmt.Sprintf("http://%s", endpoint))

	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC(services map[reflect.Type]Service) error {
	// Gather all the possible APIs to surface

	apis := n.apis()
	for _, service := range services {
		apis = append(apis, service.APIs()...)
	}

	// Start the various API endpoints, terminating all in case of errors
	if err := n.startHTTP(apis); err != nil {
		return err
	}

	// All API endpoints started successfully
	n.rpcAPIs = apis

	return nil
}

// Server retrieves the currently running P2P network layer. This method is meant
// only to inspect fields of the currently running server, life cycle management
// should be left to this Node entity.
func (n *Node) Server() *p2p.Server {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return n.server
}

// DataDir retrieves the current datadir location.
func (n *Node) DataDir() string {
	return n.config.DataDir
}

// AccountManager retrieves the account manager used by the protocol stack.
func (n *Node) GetAccountManager() *accounts.Manager {
	return n.accmgr
}

func (n *Node) GetProtocolMgr() *boy.ProtocolManager {
	return n.protocolManager
}

func (n *Node) GetDbMgr() *boydb.DatabaseManager {
	return n.dbManager
}

func (n *Node) GetWitness() []string {
	return config.WitnessList
}

func (n *Node) initDatabase() error {
	databaseDir := config.Const_DATABASE_PATH + config.Const_DATABASE_NAME
	databaseDir = path.Join(n.config.DataDir, databaseDir)

	// 初始化 数据存储
	db := boydb.GetDbInstance()
	err := db.InitDatabase(databaseDir)
	if err != nil {
		return err
	}

	n.dbManager = db

	return nil
}

func (n *Node) initGenesis() {
	pdb := memdb.GetParentMemDBInstance()
	wdb := memdb.GetWitnessMemDBInstance()

	pdb.InitParentMemDB(n.dbManager)
	wdb.InitWitnessMemDB(n.dbManager)

	genesisUnit, err := n.dbManager.GetUnitByHash(common.HexToHash(config.GENISIS_UNIT_HASH))
	if err != nil {
		genesisUnit = config.GenesisUnit()
		n.dbManager.SaveUnitToDb(genesisUnit)
		pdb.SaveNewTip(genesisUnit.Hash)
		wdb.SaveWitnessList(genesisUnit.WitnessList)
		// log.Println(genesisUnit.HashKey().String())
		// 这里存储12个见证人的未花费列表
		for i := 0; i < len(genesisUnit.WitnessList); i++ {
			unSpent := types.UTXO{UnitHash: genesisUnit.Hash, MessageIndex: 0, OutputIndex: i, Output: types.NewOutput(genesisUnit.WitnessList[i], 10000000)}
			n.dbManager.SaveUnspentOutput(genesisUnit.WitnessList[i], unSpent)

			strByte, _ := json.Marshal(unSpent)
			log.Println(genesisUnit.WitnessList[i].String(), ":", string(strByte))
		}
	}
}

// 启动节点
func (n *Node) initP2p() (*boy.ProtocolManager, error) {
	// 根据配置生成协议管理类实例
	protocol, _ := boy.NewProtocolManager(config.NETWORK_ID)

	// ECDSA算法生成密钥对
	nodeKey, err := crypto.GenerateKey()
	if err != nil {
		return &boy.ProtocolManager{}, err
	}
	port := n.config.P2P.ListenAddr

	// 获取所有协议版本
	arrProtocols := protocol.Protocol()

	// 构建 p2p Server 结构体,Server管理所有节点的连接
	n.server = &p2p.Server{
		// 服务端配置
		Config: p2p.Config{
			MaxPeers:   100,
			Name:       boy.ProtocolName,
			PrivateKey: nodeKey,
			ListenAddr: port,
			Protocols:  arrProtocols,
		},
	}

	// 遍历主网的引导节点，p2p服务器从BootstrapNodes中得到相邻节点
	for _, value := range config.MainnetBootnodes {
		nodeSuper, err := discover.ParseNode(value)

		if err != nil {
			return &boy.ProtocolManager{}, err
		}
		n.server.Config.BootstrapNodes = append(n.server.Config.BootstrapNodes, nodeSuper)
	}

	// 是否发现其他节点
	n.server.NoDiscovery = n.config.P2P.NoDiscovery

	// 启动p2p服务
	err = n.server.Start()
	if err != nil {
		log.Println(err)
		return &boy.ProtocolManager{}, err
	}

	// 给pm分配一个id
	protocol.Self = n.server.NodeInfo().ID[:8]
	protocol.StartTimer()

	log.Println("P2pServer Listening on", port)

	return protocol, nil
}

func (n *Node) Transaction() *transaction.Transaction {
	return n.transaction
}

func (n *Node) bar(count, size int64) string {
	str := ""
	for i := int64(0); i < size; i++ {
		if i < count {
			str += "="
		} else {
			str += " "
		}
	}
	return str
}

func (n *Node) showProgress(current, total int64) {
	if total == 0 {
		return
	}
	str := "[" + n.bar(int64(current)*100/total, 100) + "] " + strconv.FormatInt(current, 10) + "/" + strconv.FormatInt(total, 10)
	fmt.Printf("\r%s", str)
}

// According to the algorithm of ECDSA to generate a signature:
func (n *Node) SignMessage(data interface{}, addr common.Address, password string) (hexutil.Bytes, error) {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	account := accounts.Account{Address: addr}
	wallet, err := n.GetAccountManager().Find(account)
	if err != nil {
		return nil, err
	}
	// Assemble sign the data with the wallet
	signature, err := wallet.SignHashWithPassphrase(account, password, n.signHash(dataBytes))
	if err != nil {
		return nil, err
	}
	signature[64] += 27 // Transform V from 0/1 to 27/28 according to the yellow paper

	return signature, nil
}

// This gives context to the signed message and prevents signing of transactions.
func (n *Node) signHash(data []byte) []byte {
	msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)

	return crypto.Keccak256([]byte(msg))
}

func (n *Node) NewJoint(address string, password string, tx string, amount int) (common.Hash, error) {
	if address == "" {
		return common.Hash{}, ErrNodeSender
	} else if password == "" {
		return common.Hash{}, ErrNodePassWord
	} else if len(tx) == 0 {
		return common.Hash{}, ErrNodeAmount
	}

	// 计算要支出的总额
	if amount < 0 || amount > 100000000 {
		return common.Hash{}, ErrAmountRange
	}

	_, err := n.FindAccountWith(address)
	if err != nil {
		log.Println(err)
		return common.Hash{}, ErrNodeNoAccount
	}

	// 打包交易
	addr := common.HexToAddress(address)
	account := accounts.Account{Address: addr}

	newUnit, err := n.transaction.CreateTx(account, amount, tx, amount)
	if err != nil {
		log.Println(err)
		return common.Hash{}, err
	}

	// 签名
	signedUnit, err := n.SignMessage(newUnit, account.Address, password)
	if err != nil {
		log.Println(err)
		return common.Hash{}, ErrNodeSinged
	}
	newUnit.SetSignature(signedUnit)

	entity := types.NewUnitEntity{FromPeerId: "local", HasPeerIds: []string{}, NewUnit: newUnit}
	n.handleNewUnitEvent(entity)

	return newUnit.Hash, nil
}

// 轻节点发起一笔交易
func (n *Node) NewJointLight(address string, password string, tx string, amount int) error {
	// TODO 再增加一些安全保证检查
	if address == "" {
		return ErrNodeSender
	} else if password == "" {
		return ErrNodePassWord
	} else if len(tx) == 0 {
		return ErrNodeAmount
	}

	const ConstHeaderCommission = 100
	const ConstPayloadCommission = 100

	// 计算要支出的总额
	totalAmount := 0
	totalAmount = totalAmount + amount

	_, err := n.FindAccountWith(address)
	if err != nil {
		log.Println(err)
		return ErrNodeNoAccount
	}

	// 打包交易
	addr := common.HexToAddress(address)
	account := accounts.Account{Address: addr}

	// Calculate all spending
	totalSpend := totalAmount + ConstHeaderCommission + ConstPayloadCommission

	// 拿到该地址所有的未花费(稳定池和Pending池)
	unSpent := n.transaction.ChoiceInputs(account)

	// 这里计算出多少笔unSpent来支撑这笔输出
	// 将已经使用到unSpent记录在toOther中
	var toOther []types.UTXO
	toMyself, curAmount := 0, 0
	for _, u := range unSpent {
		curAmount = curAmount + u.Output.Amount
		toOther = append(toOther, u)
		// 当前这一笔就足够输出
		if curAmount >= totalSpend {
			//toOther = append(toOther, u)
			toMyself = curAmount - totalAmount
			// 扣除见证人手续费
			toMyself = toMyself - ConstHeaderCommission
			// 扣除挖矿手续费
			toMyself = toMyself - ConstPayloadCommission
			break
		}
	}

	// 余额不足
	if curAmount < totalAmount {
		return transaction.ErrNotEnoughBalance
	}

	// 手续费不足
	if curAmount < totalSpend {
		return transaction.ErrNotEnoughCommission
	}

	entity := types.LightNewUnitEntity{FromAddress: address}
	// TODO 临时先将单元发送到最优节点
	n.GetProtocolMgr().SendMsgToPeer(n.GetProtocolMgr().GetBestPeer(), boy.MSG_NEWUNIT_LIGHT_Q, entity)

	return nil
}

func (n *Node) CreateUnitForLight(address string, tx string, amount int) (types.Unit, error) {

	// TODO 再增加一些安全保证检查
	if address == "" {
		return types.Unit{}, ErrNodeSender
	} else if len(tx) == 0 {
		return types.Unit{}, ErrNodeAmount
	}

	// 计算要支出的总额
	outAmount := 0
	outAmount = outAmount + amount

	_, err := n.FindAccountWith(address)
	if err != nil {
		log.Println(err)
		return types.Unit{}, ErrNodeNoAccount
	}

	// 打包交易
	addr := common.HexToAddress(address)
	account := accounts.Account{Address: addr}

	newUnit, err := n.transaction.CreateTx(account, outAmount, tx, amount)
	if err != nil {
		log.Println(err)
		return types.Unit{}, ErrNodeCreateTX
	}

	// 签名这笔交易
	//newUnit.Authors = types.Authors{}
	//jsonByte, _ := json.Marshal(newUnit)
	//signedUnit, err := n.SignMessage(jsonByte, account.Address, password)
	//if err != nil {
	//	log.Println(err)
	//	return common.Hash{}, ErrNodeSinged
	//}
	//newUnit.Authors = types.Authors{types.NewAuthor(account.Address, []byte{})}
	// 计算单元的hash
	//newUnit.Hash = newUnit.HashKey()

	return newUnit, nil
}

// 通过一个地址找到指定账号
func (n *Node) FindAccountWith(address string) (accounts.Account, error) {
	if !strings.HasPrefix(address, "0x") {
		address = "0x" + address
	}
	for i := 0; i < len(n.ListAccounts()); i++ {
		if strings.EqualFold(n.ListAccounts()[i].String(), address) {
			return accounts.Account{Address: n.ListAccounts()[i]}, nil
		}
	}

	return accounts.Account{}, ErrLockAccount
}

// ListAccounts will return a list of addresses for accounts this node manages.
func (n *Node) ListAccounts() []common.Address {
	addresses := make([]common.Address, 0) // return [] instead of nil if empty
	for _, wallet := range n.GetAccountManager().Wallets() {
		for _, account := range wallet.Accounts() {
			addresses = append(addresses, account.Address)
		}
	}

	return addresses
}

func (n *Node) GetWallets() int {
	count := len(n.GetAccountManager().Wallets())

	return count
}

// NewAccount will create a new account and returns the address for the new account.
func (n *Node) NewAccount(password string) (common.Address, error) {
	acc, err := n.fetchKeystore(n.GetAccountManager()).NewAccount(password)
	if err == nil {
		return acc.Address, nil
	}

	return common.Address{}, err
}

// fetchKeystore retrives the encrypted keystore from the account manager.
func (n *Node) fetchKeystore(am *accounts.Manager) *keystore.KeyStore {

	return am.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
}

type WalletBalance struct {
	Stable  int64
	Pending map[string]int64
}

func (n *Node) GetBalance(address string) (WalletBalance, error) {

	addr := common.HexToAddress(address)
	account := accounts.Account{Address: addr}

	tran := n.Transaction()
	pendingMap := make(map[string]int64)
	walletBalance := WalletBalance{Stable: 0, Pending: pendingMap}

	unSpent := tran.FindUnspentTransactionFromStable(account.Address)
	for _, u := range unSpent {
		walletBalance.Stable = walletBalance.Stable + int64(u.Output.Amount)
	}

	unSpentPending := tran.FindUnspentTransactionFromPendingPool(account.Address)
	for _, u := range unSpentPending {
		walletBalance.Pending[u.UnitHash.String()] = int64(u.Output.Amount)
	}

	return walletBalance, nil
}

func (n *Node) PreClearCache() {
	//pdb := memdb.GetParentMemDBInstance()
	//wdb := memdb.GetWitnessMemDBInstance()
	//graphInfo := dag.NewGraphInfoGetter(n.dbManager, pdb.GetDagAllTips(), wdb.GetWitnessesAsHash())
	//startMci := graphInfo.GetLastStableBallMCI()
	//sUnits, uUnits := graphInfo.GetMissingUnits(startMci, startMci)
	//
	////log.Println("稳定点数量: ", len(sUnits), "  不稳定点数量: ", len(uUnits))
	//for _, value := range uUnits {
	//	n.dbManager.DelAllData(value.Hash.String())
	//}
}

// apis returns the collection of RPC descriptors this node offers.
func (n *Node) apis() []rpc.API {
	return []rpc.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(n),
			// todo remove flag `Public: true`
			Public: true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPublicAdminAPI(n),
			Public:    true,
		},
	}
}

// Wait blocks the thread until the node is stopped. If the node is not running
// at the time of invocation, the method immediately returns.
//func (n *Node) Wait() {
//	n.lock.RLock()
//	if n.server == nil {
//		n.lock.RUnlock()
//		return
//	}
//	stop := n.stop
//	n.lock.RUnlock()
//
//	<-stop
//}

//func (n *Node) InitDag(entity interface{}) {
//	G := entity.(types.Unit)
//
//	n.chain = make(map[common.Hash]*types.DagBlock)
//	n.DagAddUnit(G, []common.Hash{}, n.chain)
//}
//
//func (n *Node) DagAddUnit(entity interface{}, References []common.Hash, chain map[common.Hash]*types.DagBlock) {
//	unit := entity.(types.Unit)
//
//	witness := make(map[common.Hash]bool)
//	witness[unit.Hash] = false
//	for _, address := range unit.WitnessList {
//		if address == unit.Authors[0].Address {
//			witness[unit.Hash] = true
//		}
//	}
//
//	// create DagBlock
//	unitHelp := types.DagBlock{Hash: unit.Hash, Prev: make(map[common.Hash]*types.DagBlock), Next: make(map[common.Hash]*types.DagBlock), Witness: witness, WitnessLevel: unit.WitnessedLevel, Level: unit.Level}
//
//	// add references
//	for _, Reference := range References {
//		prev, ok := chain[Reference]
//		if ok {
//			unitHelp.Prev[Reference] = prev
//			prev.Next[unit.Hash] = &unitHelp
//		} else {
//			log.Println("DagAddUnit(): error! block reference invalid. block name =", unit.Hash.String(), " reference=", Reference.String())
//			os.Exit(-1)
//		}
//	}
//
//	unitHelp.SizeOfWitPastSet = n.SizeOfWitPastSet(&unitHelp)
//	log.Println("沿着最优父节点经过见证人的个数: ", unitHelp.SizeOfWitPastSet)
//	chain[unit.Hash] = &unitHelp
//}
//
//func (n *Node) SizeOfWitPastSet(unitHelp *types.DagBlock) int {
//	past := make(map[common.Hash]*types.DagBlock)
//	n.pastSet(unitHelp, past)
//	return len(past)
//}
//
//// 向前回溯,记录经过的点
//func (n *Node) pastSet(B *types.DagBlock, past map[common.Hash]*types.DagBlock) {
//	parents := make([]*types.DagBlock, 0)
//	for _, v := range B.Prev {
//		parents = append(parents, v)
//	}
//	bestParent := n.getBestParentUnit(parents)
//	if v, ok := n.chain[bestParent.Hash]; ok {
//		n.pastSet(v, past)
//	} else {
//		return
//	}
//	if bestParent.Witness[bestParent.Hash] {
//		past[bestParent.Hash] = bestParent
//	}
//}
//
//// 计算当前网络中见证人的个数
//func (n *Node) countBlue(Dag map[common.Hash]*types.DagBlock, tip types.Unit) int {
//
//	var witness = 0
//	for _, v := range Dag {
//		if wit, ok := v.Witness[tip.Hash]; ok {
//			if wit {
//				witness++
//			}
//		} else if v.Hash.String() == config.GENISIS_UNIT_HASH {
//			witness++
//		}
//	}
//
//	return witness
//}
//
//// 当前网络有多少个顶点
//func (n *Node) FindTips(G map[common.Hash]*types.DagBlock) map[common.Hash]*types.DagBlock {
//
//	tips := make(map[common.Hash]*types.DagBlock)
//
//checkNextBlock:
//	for k, v := range G {
//
//		if len(v.Next) == 0 {
//			tips[k] = v
//		} else {
//			for next := range v.Next {
//				if _, ok := G[next]; ok {
//					continue checkNextBlock
//				}
//			}
//
//			tips[v.Hash] = v
//		}
//	}
//
//	return tips
//}
//
//// 向后移动,记录经过的点
//func (n *Node) futureSet(B *types.DagBlock, future map[common.Hash]*types.DagBlock) {
//
//	for k, v := range B.Next {
//		if _, ok := future[k]; !ok {
//			n.futureSet(v, future)
//		}
//		future[k] = v
//	}
//}
//
//// 获取最优父节点
//// 最优父单元的选择策略由以下三部分组成:
//// 1. 在选择最优父单元时,见证级别最高的父单元为最优父单元
//// 2. 如果见证级别相同,则单元级别最低的作为最优父单元
//// 3. 如果两者都相同,则选择单元哈希值（sha256编码）更小的作为最优父单元
//func (n *Node) getBestParentUnit(parentList []*types.DagBlock) *types.DagBlock {
//
//	if len(parentList) == 0 {
//		return &types.DagBlock{}
//	}
//	bestParentUnit := parentList[0]
//
//	// 遍历父节点列表，更新最优父单元
//	for _, parentHash := range parentList {
//		tParentUnit := parentHash
//		// 见证等级小，更新最优
//		if bestParentUnit.WitnessLevel < tParentUnit.WitnessLevel {
//			bestParentUnit = tParentUnit
//			continue
//		}
//		// 见证等级相同，单元等级大，更新最优
//		if bestParentUnit.WitnessLevel == tParentUnit.WitnessLevel && bestParentUnit.Level > tParentUnit.Level {
//			bestParentUnit = tParentUnit
//			continue
//		}
//		// 见证等级单元等级相同,hash值最小的，更新最优
//		if bestParentUnit.WitnessLevel == tParentUnit.WitnessLevel && bestParentUnit.Level == tParentUnit.Level && bestParentUnit.Hash.String() > tParentUnit.Hash.String() {
//			bestParentUnit = tParentUnit
//			continue
//		}
//	}
//
//	return bestParentUnit
//}
