package memdb

import (
	"github.com/babyboy/leveldb"
	"github.com/babyboy/common"
	"github.com/babyboy/common/ds"
	"sync"
)

var PmDb *ParentMemDB
var oncePmDb sync.Once

// 获取父节点存储实例
func GetParentMemDBInstance() *ParentMemDB {
	oncePmDb.Do(func() {
		if PmDb == nil {
			PmDb = NewParentMemDB()
		}
	})
	return PmDb
}

type ParentMemDB struct {
	db        *boydb.DatabaseManager
	parentSet *ds.HashSet
}

//新建一个ParentMemDB
func NewParentMemDB() *ParentMemDB {
	db := &boydb.DatabaseManager{}
	parentSet := ds.NewHashSet()
	return &ParentMemDB{db, parentSet}
}

//初始化ParentMemDB
func (pdb *ParentMemDB) InitParentMemDB(db *boydb.DatabaseManager) {
	pdb.db = db
	pdb.parentSet.ListInsert(db.GetParentsList())
}

// ***父单元存储相关函数*** //

// 保存父单元（维护入度为零的点）
func (pdb *ParentMemDB) SaveParent(parent common.Hash) {
	pdb.db.SaveParent(parent)
	// 整个图中入度为零的点
	pdb.parentSet.Insert(parent)

	unit, _ := pdb.db.GetUnitByHash(parent)
	for _, val := range unit.ParentList {
		pdb.DeleteParent(val)
	}
}

// 删除父单元
func (pdb *ParentMemDB) DeleteParent(parent common.Hash) {
	pdb.db.DelParent(parent)
	pdb.parentSet.Remove(parent)
}

// 获取父单元哈希数组
func (pdb *ParentMemDB) GetParentsAsHash() []common.Hash {
	if pdb.parentSet.Empty() {
		pdb.parentSet.ListInsert(pdb.db.GetParentsList())
	}
	return pdb.parentSet.GetAllHash()
}

// 获取父单元字符串数组
func (pdb *ParentMemDB) GetParentsAsString() []string {
	if pdb.parentSet.Empty() {
		pdb.parentSet.ListInsert(pdb.db.GetParentsList())
	}
	return pdb.parentSet.GetAllHashAsString()
}

// 获取符合与单元见证列表冲突不超过1个的父节点列表
func (pdb *ParentMemDB) GetUnitParentListAsHash(witnessList []common.Address) []common.Hash {

	parents := pdb.parentSet.GetAllHash()
	witnessSet := ds.NewAddressSet()
	witnessSet.ListInsert(witnessList)
	parentList := make([]common.Hash, 0)
	//遍历父节点的见证人列表，检查是否不超过一个冲突
	for _, parent := range parents {
		tUnit, _ := pdb.db.GetUnitByHash(parent)
		count := 0
		for _, witness := range tUnit.WitnessList {
			if !witnessSet.Exists(witness) {
				count++
				if count > 1 {
					break
				}
			}
		}
		if count <= 1 {
			parentList = append(parentList, parent)
		}
	}
	return parentList
}
