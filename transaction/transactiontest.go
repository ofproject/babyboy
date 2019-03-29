package transaction

import (
	"babyboy-dag/boydb"
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/config"
	"github.com/babyboy/core/types"
	"github.com/babyboy/dag"
	"github.com/babyboy/dag/memdb"
)

func NewTransactionTest(witness int) types.Unit {

	db := boydb.GetDbInstance()
	pdb := memdb.GetParentMemDBInstance()
	wdb := memdb.GetWitnessMemDBInstance()

	gig := dag.NewGraphInfoGetter(db, pdb.GetParentsAsHash(), wdb.GetWitnessesAsHash())

	u := types.Unit{}
	u.ParentList = pdb.GetParentsAsHash()
	u.WitnessList = wdb.GetWitnessesAsHash()
	u.BestParentUnit = gig.GetBestParentUnit()
	u.Level = gig.GetLevel()
	u.WitnessedLevel = gig.GetWitnessLevel()
	u.LastBallUnit = gig.GetLastStableBall()
	u.Authors = make(types.Authors, 1)
	u.Authors[0].Address = common.HexToAddress(config.WitnessList[witness])
	u.IsStable = false
	u.MainChainIndex = 0
	u.IsOnMainChain = false
	u.SubStableMinHash = common.HexToHash("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	u.Hash = u.HashKey()

	db.SaveUnitToDb(u)
	pdb.SaveParent(u.Hash)

	mcu := dag.NewMainChainUpdater(db, u.Hash, pdb.GetParentsAsHash())

	for {
		canExtend, uints := mcu.StableBallCanExtend()
		if !canExtend {
			break
		}
		mcu.ExtendStableUnit(uints)
	}

	return u
}
