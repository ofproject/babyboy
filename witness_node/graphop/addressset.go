package graphop

import (
	"babyboy/common"
	"sync"
)

// Store the set of Address
type AddressSet struct {
	data map[common.Address]struct{}
	lock sync.RWMutex
}

// Create a new Address collection
func NewAddressSet() *AddressSet {
	return &AddressSet{data: make(map[common.Address]struct{})}
}

// Set single element insertion
func (hs *AddressSet) Insert(val common.Address) {
	hs.lock.Lock()
	defer hs.lock.Unlock()

	hs.data[val] = struct{}{}
}

// Set list insertion
func (hs *AddressSet) ListInsert(list []common.Address) {
	hs.lock.Lock()
	defer hs.lock.Unlock()

	for _, val := range list {
		hs.data[val] = struct{}{}
	}
}

// Set single element removal
func (hs *AddressSet) Remove(val common.Address) {
	hs.lock.Lock()
	defer hs.lock.Unlock()

	delete(hs.data, val)
}

// Set size
func (hs *AddressSet) Size() int {
	return len(hs.data)
}

// Whether the set is empty
func (hs *AddressSet) Empty() bool {
	return hs.Size() == 0
}

// Determine if an element exists in the set
func (hs *AddressSet) Exists(val common.Address) bool {
	hs.lock.RLock()
	defer hs.lock.RUnlock()

	_, ok := hs.data[val]
	return ok
}

// Get all the Address values in the set
func (hs *AddressSet) GetAllAddress() []common.Address {
	hs.lock.RLock()
	defer hs.lock.RUnlock()

	setList := make([]common.Address, 0)
	for val := range hs.data {
		setList = append(setList, val)
	}
	return setList
}

// Get all the Address values in the set, in the form of a string
func (hs *AddressSet) GetAllAddressAsString() []string {
	hs.lock.RLock()
	defer hs.lock.RUnlock()

	setList := make([]string, 0)
	for val := range hs.data {
		setList = append(setList, val.String())
	}
	return setList
}
