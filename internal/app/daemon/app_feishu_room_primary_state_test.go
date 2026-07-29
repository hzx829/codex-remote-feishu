package daemon

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/app/daemon/feishuroomprimary"
	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/control"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestConfigureFeishuRoomPrimaryStateMaterializesStore(t *testing.T) {
	stateDir := t.TempDir()
	store, err := feishuroomprimary.LoadStore(feishuroomprimary.StatePath(stateDir))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if err := store.Put(state.FeishuRoomPrimaryRecord{
		RoomID:           "feishu:chat:oc_room",
		ChatID:           "oc_room",
		PrimaryGatewayID: "app-1",
	}); err != nil {
		t.Fatalf("put record: %v", err)
	}

	app := New(":0", ":0", nil, agentproto.ServerIdentity{StartedAt: time.Now().UTC()})
	app.mu.Lock()
	app.configureFeishuRoomPrimaryStateLocked(stateDir)
	app.mu.Unlock()

	records := app.service.FeishuRoomPrimaryState()
	if len(records) != 1 || records[0].PrimaryGatewayID != "app-1" {
		t.Fatalf("materialized records = %#v, want app-1", records)
	}
}

func TestSyncFeishuRoomPrimaryStateDeletesClearedPrimary(t *testing.T) {
	stateDir := t.TempDir()
	store, err := feishuroomprimary.LoadStore(feishuroomprimary.StatePath(stateDir))
	if err != nil {
		t.Fatalf("load store: %v", err)
	}
	if err := store.Put(state.FeishuRoomPrimaryRecord{
		RoomID:           "feishu:chat:oc_room",
		ChatID:           "oc_room",
		PrimaryGatewayID: "app-1",
	}); err != nil {
		t.Fatalf("put record: %v", err)
	}

	app := New(":0", ":0", nil, agentproto.ServerIdentity{StartedAt: time.Now().UTC()})
	app.mu.Lock()
	app.configureFeishuRoomPrimaryStateLocked(stateDir)
	app.service.ApplySurfaceAction(control.Action{
		Kind:             control.ActionPrimaryCommand,
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ActorUserID:      "ou_user",
		Text:             "/primary off",
	})
	app.syncFeishuRoomPrimaryStateLocked()
	app.mu.Unlock()

	reloaded, err := feishuroomprimary.LoadStore(feishuroomprimary.StatePath(stateDir))
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	if _, ok := reloaded.Get("oc_room"); ok {
		t.Fatal("expected cleared primary to be deleted from store")
	}
}
