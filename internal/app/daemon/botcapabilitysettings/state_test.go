package botcapabilitysettings

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
	"github.com/kxn/codex-remote-feishu/internal/core/state"
)

func TestStoreRoundTrip(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	store, err := LoadStore(StatePath(stateDir))
	if err != nil {
		t.Fatalf("load empty store: %v", err)
	}
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))
	record := state.BotCapabilitySettingsRecord{
		GatewayID:       " app-1 ",
		ProductMode:     state.ProductModeNormal,
		Backend:         agentproto.BackendClaude,
		ClaudeProfileID: " devseek ",
		PromptOverride: state.ModelConfigRecord{
			Model:           " claude-sonnet ",
			ReasoningEffort: " MAX ",
			AccessMode:      " confirm ",
		},
		PlanMode:            state.PlanModeSettingOn,
		PlanModeOverrideSet: true,
		UpdatedBy:           " ou_user ",
		UpdatedAt:           updatedAt,
	}
	if err := store.Put(record); err != nil {
		t.Fatalf("put record: %v", err)
	}

	reloaded, err := LoadStore(StatePath(stateDir))
	if err != nil {
		t.Fatalf("reload store: %v", err)
	}
	got, ok := reloaded.Get(state.BotCapabilitySettingsKey("app-1"))
	if !ok {
		t.Fatalf("expected record after reload")
	}
	if got.GatewayID != "app-1" || got.Backend != agentproto.BackendClaude || got.ClaudeProfileID != "devseek" {
		t.Fatalf("record contract = %#v, want app-1 claude/devseek", got)
	}
	if got.PromptOverride.Model != "claude-sonnet" || got.PromptOverride.ReasoningEffort != "max" || got.PromptOverride.AccessMode != agentproto.AccessModeConfirm {
		t.Fatalf("prompt override = %#v, want normalized values", got.PromptOverride)
	}
	if got.PlanMode != state.PlanModeSettingOn || !got.PlanModeOverrideSet {
		t.Fatalf("plan = %s/%v, want on/true", got.PlanMode, got.PlanModeOverrideSet)
	}
	if got.UpdatedBy != "ou_user" || !got.UpdatedAt.Equal(updatedAt.UTC()) {
		t.Fatalf("metadata = %q/%v, want ou_user/%v", got.UpdatedBy, got.UpdatedAt, updatedAt.UTC())
	}
}

func TestLoadStoreDropsInvalidRecords(t *testing.T) {
	t.Parallel()

	stateDir := t.TempDir()
	store, err := LoadStore(StatePath(stateDir))
	if err != nil {
		t.Fatalf("load empty store: %v", err)
	}
	if err := store.Put(state.BotCapabilitySettingsRecord{
		GatewayID:   "app-1",
		ProductMode: state.ProductModeNormal,
		Backend:     agentproto.BackendCodex,
	}); err != nil {
		t.Fatalf("put valid record: %v", err)
	}
	if err := store.Put(state.BotCapabilitySettingsRecord{}); err == nil {
		t.Fatalf("expected empty gateway record to be rejected")
	}
}
