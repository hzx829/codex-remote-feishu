package state

import (
	"testing"
	"time"

	"github.com/kxn/codex-remote-feishu/internal/core/agentproto"
)

func TestNormalizeBotCapabilitySettingsRecord(t *testing.T) {
	updatedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.FixedZone("UTC+8", 8*60*60))

	record, ok := NormalizeBotCapabilitySettingsRecord(BotCapabilitySettingsRecord{
		GatewayID:       " app-1 ",
		ProductMode:     ProductModeNormal,
		Backend:         agentproto.BackendClaude,
		CodexProviderID: " team-proxy ",
		ClaudeProfileID: " devseek ",
		PromptOverride: ModelConfigRecord{
			Model:           " gpt-5.4 ",
			ReasoningEffort: " HIGH ",
			AccessMode:      " confirm ",
		},
		PlanMode:            PlanModeSettingOn,
		PlanModeOverrideSet: true,
		UpdatedBy:           " user-1 ",
		UpdatedAt:           updatedAt,
	})
	if !ok {
		t.Fatalf("expected normalized record")
	}
	if record.GatewayID != "app-1" {
		t.Fatalf("GatewayID = %q, want app-1", record.GatewayID)
	}
	if record.ProductMode != ProductModeNormal || record.Backend != agentproto.BackendClaude {
		t.Fatalf("contract = %s/%s, want normal/claude", record.ProductMode, record.Backend)
	}
	if record.CodexProviderID != "" || record.ClaudeProfileID != "devseek" {
		t.Fatalf("provider/profile = %q/%q, want empty/devseek", record.CodexProviderID, record.ClaudeProfileID)
	}
	if record.PromptOverride.Model != "gpt-5.4" || record.PromptOverride.ReasoningEffort != "high" || record.PromptOverride.AccessMode != "confirm" {
		t.Fatalf("PromptOverride = %#v, want compact normalized values", record.PromptOverride)
	}
	if record.PlanMode != PlanModeSettingOn || !record.PlanModeOverrideSet {
		t.Fatalf("plan override = %s/%v, want on/true", record.PlanMode, record.PlanModeOverrideSet)
	}
	if record.UpdatedBy != "user-1" {
		t.Fatalf("UpdatedBy = %q, want user-1", record.UpdatedBy)
	}
	if record.UpdatedAt.Location() != time.UTC {
		t.Fatalf("UpdatedAt location = %v, want UTC", record.UpdatedAt.Location())
	}
}

func TestBotCapabilitySettingsKeyRequiresGateway(t *testing.T) {
	if key := BotCapabilitySettingsKey(" app-1 "); key != "feishu:gateway:app-1" {
		t.Fatalf("key = %q, want feishu:gateway:app-1", key)
	}
	if _, ok := NormalizeBotCapabilitySettingsRecord(BotCapabilitySettingsRecord{}); ok {
		t.Fatalf("expected empty gateway record to be rejected")
	}
}

func TestEffectiveSurfaceCapabilitySettingsUsesBotRecordForFeishuRoom(t *testing.T) {
	root := NewRoot()
	root.BotCapabilitySettings["feishu:gateway:app-1"] = BotCapabilitySettingsRecord{
		GatewayID:           "app-1",
		ProductMode:         ProductModeNormal,
		Backend:             agentproto.BackendClaude,
		ClaudeProfileID:     "devseek",
		PromptOverride:      ModelConfigRecord{Model: "claude-sonnet", ReasoningEffort: "max"},
		PlanMode:            PlanModeSettingOn,
		PlanModeOverrideSet: true,
	}
	surface := &SurfaceConsoleRecord{
		SurfaceSessionID: "feishu:app-1:chat:oc_room",
		Platform:         "feishu",
		GatewayID:        "app-1",
		ChatID:           "oc_room",
		ProductMode:      ProductModeNormal,
		Backend:          agentproto.BackendCodex,
		CodexProviderID:  "team-proxy",
		PromptOverride:   ModelConfigRecord{Model: "gpt-5.4"},
		PlanMode:         PlanModeSettingOff,
	}

	effective := EffectiveSurfaceCapabilitySettings(root, surface)
	if effective.Contract.Backend != agentproto.BackendClaude || effective.Contract.ClaudeProfileID != "devseek" {
		t.Fatalf("effective contract = %#v, want bot claude profile", effective.Contract)
	}
	if effective.PromptOverride.Model != "claude-sonnet" || effective.PromptOverride.ReasoningEffort != "max" {
		t.Fatalf("effective prompt = %#v, want bot prompt", effective.PromptOverride)
	}
	if effective.PlanMode != PlanModeSettingOn || !effective.PlanModeOverrideSet {
		t.Fatalf("effective plan = %s/%v, want bot on/true", effective.PlanMode, effective.PlanModeOverrideSet)
	}
}

func TestEffectiveSurfaceCapabilitySettingsKeepsPrivateSurfaceLocal(t *testing.T) {
	root := NewRoot()
	root.BotCapabilitySettings["feishu:gateway:app-1"] = BotCapabilitySettingsRecord{
		GatewayID:       "app-1",
		ProductMode:     ProductModeNormal,
		Backend:         agentproto.BackendClaude,
		ClaudeProfileID: "devseek",
	}
	surface := &SurfaceConsoleRecord{
		SurfaceSessionID: "feishu:app-1:user:ou_user",
		Platform:         "feishu",
		GatewayID:        "app-1",
		ChatID:           "ou_user",
		ActorUserID:      "ou_user",
		ProductMode:      ProductModeNormal,
		Backend:          agentproto.BackendCodex,
		CodexProviderID:  "team-proxy",
	}

	effective := EffectiveSurfaceCapabilitySettings(root, surface)
	if effective.Contract.Backend != agentproto.BackendCodex || effective.Contract.CodexProviderID != "team-proxy" {
		t.Fatalf("effective contract = %#v, want private surface local contract", effective.Contract)
	}
}
