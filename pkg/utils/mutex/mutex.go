package mutex

import (
	"sync"
	"sync/atomic"
)

const (
	mutexUnlocked int32 = 0
	mutexLocked   int32 = 1
)

// OptimMutex is an unbocked optimistic mutex.
type OptimMutex interface {
	TryLock() bool
	TryUnlock() bool
	IsLocked() bool
}

// mutex is an unbocked optimistic mutex.
type mutex struct {
	state *int32
}

// TryLock returns true if locks successfully.
func (m *mutex) TryLock() bool {
	return atomic.CompareAndSwapInt32(m.state, mutexUnlocked, mutexLocked)
}

// TryUnlock returns true if unlocks successfully.
func (m *mutex) TryUnlock() bool {
	return atomic.CompareAndSwapInt32(m.state, mutexLocked, mutexUnlocked)
}

// IsLocked returns true if the mutex is locked.
func (m *mutex) IsLocked() bool {
	return atomic.LoadInt32(m.state) == mutexLocked
}

// NewOptimMutex returns an unbocked optimistic mutex.
func NewOptimMutex() OptimMutex {
	state := mutexUnlocked
	return &mutex{
		state: &state,
	}
}

// GroupMutex manages a group of mutexes.
type GroupMutex interface {
	Lock(string) bool
	Unlock(string) bool
	DelMutex(string)
}

type groupMutex struct {
	m     sync.Mutex
	mxMap map[string]OptimMutex
}

// Lock returns true if mutex with key locks successfully, else returns false.
func (cm *groupMutex) Lock(key string) bool {
	cm.m.Lock()
	defer cm.m.Unlock()
	mx, ok := cm.mxMap[key]
	if !ok {
		mx = NewOptimMutex()
		cm.mxMap[key] = mx
	}
	return mx.TryLock()
}

// Unlock unlocks the mutex with key.
func (cm *groupMutex) Unlock(key string) bool {
	cm.m.Lock()
	defer cm.m.Unlock()
	mx, ok := cm.mxMap[key]
	if !ok {
		return false
	}
	return mx.TryUnlock()
}

// DelMutex removes the mutex with key.
func (cm *groupMutex) DelMutex(key string) {
	cm.m.Lock()
	defer cm.m.Unlock()
	delete(cm.mxMap, key)
}

// NewGroupMutex returns a new group mutex.
func NewGroupMutex() GroupMutex {
	return &groupMutex{
		mxMap: map[string]OptimMutex{},
	}
}
