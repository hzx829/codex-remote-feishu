package daemon

import (
	"log"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/feishuroomprimary"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func (a *App) configureFeishuRoomPrimaryStateLocked(stateDir string) {
	path := feishuroomprimary.StatePath(stateDir)
	a.feishuRoomPrimaryState.persistedStoreRuntimeState = loadPersistedStore("Feishu room primary", path, feishuroomprimary.LoadStore)
	a.materializeFeishuRoomPrimaryStateLocked()
}

func (a *App) materializeFeishuRoomPrimaryStateLocked() {
	if a.feishuRoomPrimaryState.store == nil {
		return
	}
	entries := a.feishuRoomPrimaryState.store.Entries()
	records := make([]state.FeishuRoomPrimaryRecord, 0, len(entries))
	for _, record := range entries {
		records = append(records, record)
	}
	a.service.MaterializeFeishuRoomPrimaryState(records)
}

func (a *App) syncFeishuRoomPrimaryStateLocked() {
	if !a.feishuRoomPrimaryState.writable() || a.feishuRoomPrimaryState.store == nil {
		return
	}
	desired := map[string]bool{}
	for _, record := range a.service.FeishuRoomPrimaryState() {
		key := state.FeishuRoomPrimaryKey(record.RoomID)
		if key == "" {
			continue
		}
		desired[key] = true
		if err := a.feishuRoomPrimaryState.store.Put(record); err != nil {
			log.Printf("persist feishu room primary state failed: room=%s err=%v", record.RoomID, err)
		}
	}
	for key := range a.feishuRoomPrimaryState.store.Entries() {
		if desired[key] {
			continue
		}
		if err := a.feishuRoomPrimaryState.store.Delete(key); err != nil {
			log.Printf("delete feishu room primary state failed: room=%s err=%v", key, err)
		}
	}
}
