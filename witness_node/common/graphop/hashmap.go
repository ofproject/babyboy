package ds

import (
	"sync"

	"babyboy/common"
	"babyboy/core/types"
)

// a map that stores element hashes and cell contents HashMap struct {
type HashMap struct {
	data map[common.Hash]types.Unit
	lock sync.RWMutex
}

// Create a new Hashmap
func NewHashMap() *HashMap {
	return &HashMap{data: make(map[common.Hash]types.Unit)}
}

// Map single element insertion
func (hm *HashMap) Insert(h common.Hash, u types.Unit) {
	hm.lock.Lock()
	defer hm.lock.Unlock()
	hm.data[h] = u
}

// Map single element removal
func (hm *HashMap) Remove(h common.Hash) {
	hm.lock.Lock()
	defer hm.lock.Unlock()

	delete(hm.data, h)
}

// return map size
func (hm *HashMap) Size() int {
	return len(hm.data)

}

//  return whether the map is empty
func (hm *HashMap) Empty() bool {
	return hm.Size() == 0
}

// Read the elements corresponding to Hash from the map
func (hm *HashMap) Read(h common.Hash) types.Unit {
	hm.lock.RLock()
	defer hm.lock.RUnlock()

	return hm.data[h]
}

// Determine if there is a val in the map
func (hm *HashMap) Exists(val common.Hash) bool {
	hm.lock.RLock()
	defer hm.lock.RUnlock()

	_, ok := hm.data[val]
	return ok
}

// Get all the data in the map
func (hm *HashMap) GetAllData() map[common.Hash]types.Unit {
	hm.lock.RLock()
	defer hm.lock.RUnlock()
	return hm.data
}

// Get all the values in the map
func (hm *HashMap) GetAllValue() types.Units {
	hm.lock.RLock()
	defer hm.lock.RUnlock()
	list := make(types.Units, 0)
	for _, val := range hm.data {
		list = append(list, val)
	}
	return list
}

// Get all the keys of the map
func (hm *HashMap) GetAllKey() []common.Hash {
	hm.lock.RLock()
	defer hm.lock.RUnlock()

	mapList := make([]common.Hash, 0)
	for val := range hm.data {
		mapList = append(mapList, val)
	}
	return mapList
}

// Get map all keys in the form of a string
func (hm *HashMap) GetAllKeyAsString() []string {
	hm.lock.RLock()
	defer hm.lock.RUnlock()

	mapList := make([]string, 0)
	for val := range hm.data {
		mapList = append(mapList, val.String())
	}
	return mapList
}
