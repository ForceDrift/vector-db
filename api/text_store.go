package main

import (
	"encoding/json"
	"os"
	"sync"
)

type TextStore struct {
	mu    sync.RWMutex
	texts map[uint64]string
	path  string
}

func NewTextStore(path string) *TextStore {
	ts := &TextStore{
		texts: make(map[uint64]string),
		path:  path,
	}
	ts.load()
	return ts
}

func (ts *TextStore) load() {
	data, err := os.ReadFile(ts.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &ts.texts)
}

func (ts *TextStore) save() {
	data, _ := json.Marshal(ts.texts)
	os.MkdirAll(ts.path[:len(ts.path)-len("/texts.json")], 0755)
	os.WriteFile(ts.path, data, 0644)
}

func (ts *TextStore) Set(id uint64, text string) {
	ts.mu.Lock()
	ts.texts[id] = text
	ts.mu.Unlock()
	ts.save()
}

func (ts *TextStore) Get(id uint64) (string, bool) {
	ts.mu.RLock()
	t, ok := ts.texts[id]
	ts.mu.RUnlock()
	return t, ok
}

func (ts *TextStore) GetBatch(ids []uint64) map[uint64]string {
	out := make(map[uint64]string)
	ts.mu.RLock()
	for _, id := range ids {
		if t, ok := ts.texts[id]; ok {
			out[id] = t
		}
	}
	ts.mu.RUnlock()
	return out
}
