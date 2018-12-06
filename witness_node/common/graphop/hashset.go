package ds

import (
	"babyboy-dag/common"
	"sync"
)

type HashSet struct {
	data map[common.Hash]struct{}
	lock sync.RWMutex
}

func NewHashSet() *HashSet {
	return &HashSet{data: make(map[common.Hash]struct{})}
}

func (hs *HashSet) Insert(val common.Hash) {
	hs.lock.Lock()
	defer hs.lock.Unlock()

	hs.data[val] = struct{}{}
}

func (hs *HashSet) ListInsert(list []common.Hash) {
	hs.lock.Lock()
	defer hs.lock.Unlock()

	for _, val := range list {
		hs.data[val] = struct{}{}
	}
}

func (hs *HashSet) Remove(val common.Hash) {
	hs.lock.Lock()
	defer hs.lock.Unlock()

	delete(hs.data, val)
}

func (hs *HashSet) Size() int {
	return len(hs.data)
}

func (hs *HashSet) Empty() bool {
	return hs.Size() == 0
}

func (hs *HashSet) Exists(val common.Hash) bool {
	hs.lock.RLock()
	defer hs.lock.RUnlock()

	_, ok := hs.data[val]
	return ok
}

func (hs *HashSet) GetAllHash() []common.Hash {
	hs.lock.RLock()
	defer hs.lock.RUnlock()

	setList := make([]common.Hash, 0)
	for val := range hs.data {
		setList = append(setList, val)
	}
	return setList
}

func (hs *HashSet) GetAllHashAsString() []string {
	hs.lock.RLock()
	defer hs.lock.RUnlock()

	setList := make([]string, 0)
	for val := range hs.data {
		setList = append(setList, val.String())
	}
	return setList
}
