package daemon

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/botcapabilitysettings"
	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestConfigureBotCapabilitySettingsStateMaterializesStore(t *testing.T) {
	stateDir := t.TempDir()
	store, err := botcapabilitysettings.LoadStore(botcapabilitysettings.StatePath(stateDir))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if err := store.Put(state.BotCapabilitySettingsRecord{
		GatewayID:       "app-1",
		ProductMode:     state.ProductModeNormal,
		Backend:         agentproto.BackendClaude,
		ClaudeProfileID: "devseek",
	}); err != nil {
		t.Fatalf("put record: %v", err)
	}

	app := New(":0", ":0", nil, agentproto.ServerIdentity{StartedAt: time.Now().UTC()})
	app.mu.Lock()
	app.configureBotCapabilitySettingsStateLocked(stateDir)
	app.service.MaterializeSurfaceResumeWithCodexProvider(
		"feishu:app-1:chat:oc_room",
		"app-1",
		"oc_room",
		"ou_user",
		state.ProductModeNormal,
		agentproto.BackendCodex,
		"team-proxy",
		"",
		state.SurfaceVerbosityNormal,
		state.PlanModeSettingOff,
	)
	app.mu.Unlock()

	if got := app.service.SurfaceBackend("feishu:app-1:chat:oc_room"); got != agentproto.BackendClaude {
		t.Fatalf("SurfaceBackend = %s, want claude", got)
	}
}

func TestSyncBotCapabilitySettingsStatePersistsPrivateCommand(t *testing.T) {
	stateDir := t.TempDir()
	app := New(":0", ":0", nil, agentproto.ServerIdentity{StartedAt: time.Now().UTC()})

	app.mu.Lock()
	app.configureBotCapabilitySettingsStateLocked(stateDir)
	app.service.ApplySurfaceAction(control.Action{
		Kind:             control.ActionModeCommand,
		SurfaceSessionID: "feishu:app-1:user:ou_user",
		GatewayID:        "app-1",
		ChatID:           "ou_user",
		ActorUserID:      "ou_user",
		Text:             "/mode claude",
	})
	app.syncBotCapabilitySettingsStateLocked()
	app.mu.Unlock()

	reloaded, err := botcapabilitysettings.LoadStore(botcapabilitysettings.StatePath(stateDir))
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	record, ok := reloaded.Get(state.BotCapabilitySettingsKey("app-1"))
	if !ok {
		t.Fatalf("expected persisted bot capability settings")
	}
	if record.Backend != agentproto.BackendClaude {
		t.Fatalf("persisted backend = %s, want claude", record.Backend)
	}
}
