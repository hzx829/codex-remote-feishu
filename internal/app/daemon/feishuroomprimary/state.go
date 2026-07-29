package feishuroomprimary

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

const (
	StateVersion  = 1
	StateFileName = "feishu-room-primary.json"
)

type StateFile struct {
	Version int                                      `json:"version"`
	Entries map[string]state.FeishuRoomPrimaryRecord `json:"entries,omitempty"`
}

type Store struct {
	path    string
	entries map[string]state.FeishuRoomPrimaryRecord
	dirty   bool
}

func StatePath(stateDir string) string {
	stateDir = strings.TrimSpace(stateDir)
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, StateFileName)
}

func NewStore(path string) *Store {
	return &Store{
		path:    strings.TrimSpace(path),
		entries: map[string]state.FeishuRoomPrimaryRecord{},
	}
}

func LoadStore(path string) (*Store, error) {
	store := NewStore(path)
	if store.path == "" {
		return store, nil
	}
	raw, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, err
	}
	var persisted StateFile
	if err := json.Unmarshal(raw, &persisted); err != nil {
		return nil, err
	}
	if persisted.Version == 0 {
		persisted.Version = StateVersion
	}
	if persisted.Version != StateVersion {
		return nil, fmt.Errorf("unsupported feishu room primary state version: %d", persisted.Version)
	}
	for key, record := range persisted.Entries {
		normalized, ok := state.NormalizeFeishuRoomPrimaryRecord(record)
		if !ok {
			store.dirty = true
			continue
		}
		canonicalKey := state.FeishuRoomPrimaryKey(normalized.RoomID)
		if canonicalKey == "" {
			store.dirty = true
			continue
		}
		if strings.TrimSpace(key) != canonicalKey {
			store.dirty = true
		}
		store.entries[canonicalKey] = normalized
	}
	return store, nil
}

func (s *Store) Entries() map[string]state.FeishuRoomPrimaryRecord {
	if s == nil || len(s.entries) == 0 {
		return map[string]state.FeishuRoomPrimaryRecord{}
	}
	values := make(map[string]state.FeishuRoomPrimaryRecord, len(s.entries))
	for key, record := range s.entries {
		values[key] = record
	}
	return values
}

func (s *Store) Get(key string) (state.FeishuRoomPrimaryRecord, bool) {
	if s == nil {
		return state.FeishuRoomPrimaryRecord{}, false
	}
	record, ok := s.entries[state.FeishuRoomPrimaryKey(key)]
	if !ok {
		return state.FeishuRoomPrimaryRecord{}, false
	}
	return record, true
}

func (s *Store) Put(record state.FeishuRoomPrimaryRecord) error {
	if s == nil {
		return nil
	}
	normalized, ok := state.NormalizeFeishuRoomPrimaryRecord(record)
	if !ok {
		return fmt.Errorf("feishu room primary state requires room identity")
	}
	key := state.FeishuRoomPrimaryKey(normalized.RoomID)
	if key == "" {
		return fmt.Errorf("feishu room primary state requires room identity")
	}
	s.entries[key] = normalized
	return s.Save()
}

func (s *Store) Delete(key string) error {
	if s == nil {
		return nil
	}
	key = state.FeishuRoomPrimaryKey(key)
	if key == "" {
		return nil
	}
	delete(s.entries, key)
	return s.Save()
}

func (s *Store) Save() error {
	if s == nil || s.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	persisted := StateFile{
		Version: StateVersion,
		Entries: s.Entries(),
	}
	raw, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmpFile, err := os.CreateTemp(filepath.Dir(s.path), filepath.Base(s.path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if err := tmpFile.Chmod(0o600); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if _, err := tmpFile.Write(raw); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, s.path); err != nil {
		return err
	}
	s.dirty = false
	return nil
}

func (s *Store) Dirty() bool {
	if s == nil {
		return false
	}
	return s.dirty
}
