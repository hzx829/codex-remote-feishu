package feishuroomprimary

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	store, err := LoadStore(StatePath(stateDir))
	if err != nil {
		t.Fatalf("load empty store: %v", err)
	}
	updatedAt := time.Date(2026, 7, 29, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	if err := store.Put(state.FeishuRoomPrimaryRecord{
		RoomID:           " feishu:chat:oc_room ",
		ChatID:           " oc_room ",
		PrimaryGatewayID: " app-1 ",
		PrimaryUpdatedBy: " ou_user ",
		PrimaryUpdatedAt: updatedAt,
	}); err != nil {
		t.Fatalf("put record: %v", err)
	}

	reloaded, err := LoadStore(StatePath(stateDir))
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	got, ok := reloaded.Get(state.FeishuRoomPrimaryKey("oc_room"))
	if !ok {
		t.Fatal("expected record after reload")
	}
	if got.RoomID != "feishu:chat:oc_room" || got.ChatID != "oc_room" || got.PrimaryGatewayID != "app-1" {
		t.Fatalf("record identity = %#v, want normalized room/chat/gateway", got)
	}
	if got.PrimaryUpdatedBy != "ou_user" || !got.PrimaryUpdatedAt.Equal(updatedAt.UTC()) {
		t.Fatalf("metadata = %#v, want UTC-normalized update metadata", got)
	}
}

func TestLoadStoreDropsInvalidRecords(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	store, err := LoadStore(StatePath(stateDir))
	if err != nil {
		t.Fatalf("load empty store: %v", err)
	}
	if err := store.Put(state.FeishuRoomPrimaryRecord{
		RoomID:           "feishu:chat:oc_room",
		ChatID:           "oc_room",
		PrimaryGatewayID: "app-1",
	}); err != nil {
		t.Fatalf("put valid record: %v", err)
	}
	if err := store.Put(state.FeishuRoomPrimaryRecord{}); err == nil {
		t.Fatal("expected empty room primary record to be rejected")
	}
}
