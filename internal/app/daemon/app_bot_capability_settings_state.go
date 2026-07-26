package daemon

import (
	"log"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/botcapabilitysettings"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func (a *App) configureBotCapabilitySettingsStateLocked(stateDir string) {
	path := botcapabilitysettings.StatePath(stateDir)
	store, err := botcapabilitysettings.LoadStore(path)
	if err != nil {
		log.Printf("load bot capability settings state failed: path=%s err=%v", path, err)
		store = botcapabilitysettings.NewStore(path)
	}
	if store != nil && store.Dirty() {
		if err := store.Save(); err != nil {
			log.Printf("persist sanitized bot capability settings state failed: path=%s err=%v", path, err)
		}
	}
	a.botCapabilitySettingsState.store = store
	a.materializeBotCapabilitySettingsStateLocked()
}

func (a *App) materializeBotCapabilitySettingsStateLocked() {
	if a.botCapabilitySettingsState.store == nil {
		return
	}
	entries := a.botCapabilitySettingsState.store.Entries()
	records := make([]state.BotCapabilitySettingsRecord, 0, len(entries))
	for _, record := range entries {
		records = append(records, record)
	}
	a.service.MaterializeBotCapabilitySettings(records)
}

func (a *App) syncBotCapabilitySettingsStateLocked() {
	if a.botCapabilitySettingsState.store == nil {
		return
	}
	for _, record := range a.service.BotCapabilitySettings() {
		if err := a.botCapabilitySettingsState.store.Put(record); err != nil {
			log.Printf("persist bot capability settings failed: gateway=%s err=%v", record.GatewayID, err)
		}
	}
}
