package plugin

import (
	"sync"
)

type taskStore struct {
	store map[string]*taskHandle
	mu    sync.RWMutex
}

func newTaskStore() *taskStore {
	return &taskStore{
		store: make(map[string]*taskHandle),
	}
}

func (s *taskStore) Put(taskID string, handle *taskHandle) {
	s.mu.Lock()
	s.store[taskID] = handle
	s.mu.Unlock()
}

func (s *taskStore) Delete(taskID string) {
	s.mu.Lock()
	delete(s.store, taskID)
	s.mu.Unlock()
}

func (s *taskStore) Get(taskID string) (*taskHandle, bool) {
	s.mu.RLock()
	th, ok := s.store[taskID]
	s.mu.RUnlock()
	return th, ok
}
